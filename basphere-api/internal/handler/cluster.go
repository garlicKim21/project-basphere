package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/basphere/basphere-api/internal/model"
)

// Cluster API handlers

// apiCreateCluster handles POST /api/v1/clusters
func (h *Handler) apiCreateCluster(w http.ResponseWriter, r *http.Request) {
	// Get username from header
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "Missing X-Basphere-User header")
		return
	}

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered")
		return
	}

	// Parse input
	var input model.CreateClusterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.jsonError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate input
	if errors := input.Validate(); len(errors) > 0 {
		h.jsonError(w, http.StatusBadRequest, "Validation failed", errors...)
		return
	}

	// Check quota
	quota, err := h.provisioner.GetClusterQuota(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to get quota", err.Error())
		return
	}
	if quota.UsedClusters >= quota.MaxClusters {
		h.jsonError(w, http.StatusForbidden, "Cluster quota exceeded")
		return
	}

	// Check if cluster already exists
	clusterExists, err := h.provisioner.ClusterExists(username, input.Name)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check cluster", err.Error())
		return
	}
	if clusterExists {
		h.jsonError(w, http.StatusConflict, "Cluster already exists")
		return
	}

	// Create cluster
	cluster, err := h.provisioner.CreateCluster(username, &input)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to create cluster", err.Error())
		return
	}

	h.jsonSuccess(w, "Cluster creation started", cluster)
}

// apiListClusters handles GET /api/v1/clusters
func (h *Handler) apiListClusters(w http.ResponseWriter, r *http.Request) {
	// Get username from header
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "Missing X-Basphere-User header")
		return
	}

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered")
		return
	}

	// List clusters
	clusters, err := h.provisioner.ListClusters(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to list clusters", err.Error())
		return
	}

	// Get quota
	quota, err := h.provisioner.GetClusterQuota(username)
	if err != nil {
		quota = &model.ClusterQuota{} // Default empty quota on error
	}

	response := model.ClusterListResponse{
		Clusters: clusters,
		Total:    len(clusters),
		Quota:    *quota,
	}

	h.jsonSuccess(w, "", response)
}

// apiGetCluster handles GET /api/v1/clusters/{name}
func (h *Handler) apiGetCluster(w http.ResponseWriter, r *http.Request) {
	// Get username from header
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "Missing X-Basphere-User header")
		return
	}

	clusterName := chi.URLParam(r, "name")

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered")
		return
	}

	// Get cluster
	cluster, err := h.provisioner.GetCluster(username, clusterName)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "Cluster not found", err.Error())
		return
	}

	h.jsonSuccess(w, "", cluster)
}

// apiDeleteCluster handles DELETE /api/v1/clusters/{name}
func (h *Handler) apiDeleteCluster(w http.ResponseWriter, r *http.Request) {
	// Get username from header
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "Missing X-Basphere-User header")
		return
	}

	clusterName := chi.URLParam(r, "name")

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered")
		return
	}

	// Check if cluster exists
	clusterExists, err := h.provisioner.ClusterExists(username, clusterName)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check cluster", err.Error())
		return
	}
	if !clusterExists {
		h.jsonError(w, http.StatusNotFound, "Cluster not found")
		return
	}

	// Delete cluster
	if err := h.provisioner.DeleteCluster(username, clusterName); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to delete cluster", err.Error())
		return
	}

	h.jsonSuccess(w, "Cluster deletion started", nil)
}

// apiGetKubeconfig handles GET /api/v1/clusters/{name}/kubeconfig
func (h *Handler) apiGetKubeconfig(w http.ResponseWriter, r *http.Request) {
	// Get username from header
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "Missing X-Basphere-User header")
		return
	}

	clusterName := chi.URLParam(r, "name")

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered")
		return
	}

	// Check if cluster exists
	clusterExists, err := h.provisioner.ClusterExists(username, clusterName)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check cluster", err.Error())
		return
	}
	if !clusterExists {
		h.jsonError(w, http.StatusNotFound, "Cluster not found")
		return
	}

	// Check for refresh parameter
	refresh := r.URL.Query().Get("refresh") == "true"
	_ = refresh // TODO: Use refresh parameter to force kubeconfig extraction

	// Get kubeconfig
	kubeconfig, err := h.provisioner.GetKubeconfig(username, clusterName)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to get kubeconfig", err.Error())
		return
	}

	h.jsonSuccess(w, "", model.KubeconfigResponse{
		Kubeconfig: string(kubeconfig),
	})
}

// apiGetClusterStatus handles GET /api/v1/clusters/{name}/status
func (h *Handler) apiGetClusterStatus(w http.ResponseWriter, r *http.Request) {
	// Get username from header
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "Missing X-Basphere-User header")
		return
	}

	clusterName := chi.URLParam(r, "name")

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered")
		return
	}

	// Get cluster status
	cluster, err := h.provisioner.GetCluster(username, clusterName)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "Cluster not found", err.Error())
		return
	}

	response := model.ClusterStatusResponse{
		Name:   cluster.Name,
		Status: cluster.Status,
	}

	h.jsonSuccess(w, "", response)
}

// apiGetClusterQuota handles GET /api/v1/clusters/quota
func (h *Handler) apiGetClusterQuota(w http.ResponseWriter, r *http.Request) {
	// Get username from header
	username := r.Header.Get("X-Basphere-User")
	if username == "" {
		h.jsonError(w, http.StatusUnauthorized, "Missing X-Basphere-User header")
		return
	}

	// Check if user exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if !exists {
		h.jsonError(w, http.StatusForbidden, "User not registered")
		return
	}

	// Get quota
	quota, err := h.provisioner.GetClusterQuota(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to get quota", err.Error())
		return
	}

	h.jsonSuccess(w, "", quota)
}
