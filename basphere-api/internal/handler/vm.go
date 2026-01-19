package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/basphere/basphere-api/internal/model"
)

// VM API handlers

// apiCreateVM handles POST /api/v1/vms
func (h *Handler) apiCreateVM(w http.ResponseWriter, r *http.Request) {
	// Get username from header (set by CLI)
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "X-Basphere-User header required")
		return
	}

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered", username)
		return
	}

	// Parse input
	var input model.CreateVMInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.jsonError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate input
	if errors := input.Validate(); len(errors) > 0 {
		h.jsonError(w, http.StatusBadRequest, "Validation failed", errors...)
		return
	}

	// Default count to 1
	if input.Count <= 0 {
		input.Count = 1
	}

	// Check quota
	quota, err := h.provisioner.GetQuota(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to get quota", err.Error())
		return
	}

	if quota.UsedVMs+input.Count > quota.MaxVMs {
		h.jsonError(w, http.StatusForbidden, "VM quota exceeded",
			"current: "+string(rune(quota.UsedVMs+'0'))+", requested: "+string(rune(input.Count+'0'))+", max: "+string(rune(quota.MaxVMs+'0')))
		return
	}

	// Check if VM already exists (for single VM)
	if input.Count == 1 {
		vmExists, err := h.provisioner.VMExists(username, input.Name)
		if err != nil {
			h.jsonError(w, http.StatusInternalServerError, "Failed to check VM", err.Error())
			return
		}
		if vmExists {
			h.jsonError(w, http.StatusConflict, "VM already exists", input.Name)
			return
		}
	}

	// Create VMs
	var createdVMs []model.VM
	var errors []string
	created := 0
	failed := 0

	for i := 1; i <= input.Count; i++ {
		vmName := input.Name
		if input.Count > 1 {
			vmName = input.Name + "-" + string(rune(i+'0'-1))
		}

		vmInput := &model.CreateVMInput{
			Name: vmName,
			OS:   input.OS,
			Spec: input.Spec,
		}

		vm, err := h.provisioner.CreateVM(username, vmInput)
		if err != nil {
			failed++
			errors = append(errors, "Failed to create "+vmName+": "+err.Error())
			continue
		}

		createdVMs = append(createdVMs, *vm)
		created++
	}

	// Return response
	resp := model.CreateVMResponse{
		VMs:     createdVMs,
		Created: created,
		Failed:  failed,
		Errors:  errors,
	}

	if created == 0 {
		h.jsonResponse(w, http.StatusInternalServerError, apiResponse{
			Success: false,
			Message: "Failed to create VMs",
			Data:    resp,
		})
		return
	}

	h.jsonSuccess(w, "VMs created", resp)
}

// apiListVMs handles GET /api/v1/vms
func (h *Handler) apiListVMs(w http.ResponseWriter, r *http.Request) {
	// Get username from header
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "X-Basphere-User header required")
		return
	}

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered", username)
		return
	}

	// List VMs
	vms, err := h.provisioner.ListVMs(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to list VMs", err.Error())
		return
	}

	// Get quota
	quota, err := h.provisioner.GetQuota(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to get quota", err.Error())
		return
	}

	resp := model.VMListResponse{
		VMs:   vms,
		Total: len(vms),
		Quota: *quota,
	}

	h.jsonSuccess(w, "", resp)
}

// apiGetVM handles GET /api/v1/vms/{name}
func (h *Handler) apiGetVM(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "X-Basphere-User header required")
		return
	}

	vmName := chi.URLParam(r, "name")
	if vmName == "" {
		h.jsonError(w, http.StatusBadRequest, "VM name required")
		return
	}

	vm, err := h.provisioner.GetVM(username, vmName)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "VM not found", err.Error())
		return
	}

	h.jsonSuccess(w, "", vm)
}

// apiDeleteVM handles DELETE /api/v1/vms/{name}
func (h *Handler) apiDeleteVM(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "X-Basphere-User header required")
		return
	}

	vmName := chi.URLParam(r, "name")
	if vmName == "" {
		h.jsonError(w, http.StatusBadRequest, "VM name required")
		return
	}

	// Check if VM exists
	vm, err := h.provisioner.GetVM(username, vmName)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "VM not found", err.Error())
		return
	}

	// Delete VM
	if err := h.provisioner.DeleteVM(username, vmName); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to delete VM", err.Error())
		return
	}

	h.jsonSuccess(w, "VM deleted", vm)
}

// apiGetQuota handles GET /api/v1/quota
func (h *Handler) apiGetQuota(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "X-Basphere-User header required")
		return
	}

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered", username)
		return
	}

	quota, err := h.provisioner.GetQuota(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to get quota", err.Error())
		return
	}

	h.jsonSuccess(w, "", quota)
}
