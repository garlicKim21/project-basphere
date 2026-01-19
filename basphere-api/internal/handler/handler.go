package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/basphere/basphere-api/internal/config"
	"github.com/basphere/basphere-api/internal/model"
	"github.com/basphere/basphere-api/internal/provisioner"
	"github.com/basphere/basphere-api/internal/store"
)

// Handler handles HTTP requests
type Handler struct {
	store          store.Store
	keyChangeStore *store.KeyChangeStore
	provisioner    provisioner.Provisioner
	templates      *template.Template
	config         *config.Config
}

// NewHandler creates a new handler
func NewHandler(s store.Store, prov provisioner.Provisioner, templateDir string, cfg *config.Config) (*Handler, error) {
	tmpl, err := template.ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		// Templates might not exist yet, that's okay
		log.Printf("Warning: failed to parse templates: %v", err)
		tmpl = template.New("empty")
	}

	// Initialize key change store in the same base directory
	keyChangeStore, err := store.NewKeyChangeStore(cfg.Storage.PendingDir)
	if err != nil {
		log.Printf("Warning: failed to initialize key change store: %v", err)
	}

	return &Handler{
		store:          s,
		keyChangeStore: keyChangeStore,
		provisioner:    prov,
		templates:      tmpl,
		config:         cfg,
	}, nil
}

// Router returns the HTTP router
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// Web routes (HTML)
	r.Get("/", h.indexPage)
	r.Get("/register", h.registerPage)
	r.Post("/register", h.registerFormSubmit)
	r.Get("/success", h.successPage)
	r.Get("/ssh-guide", h.sshGuidePage)
	r.Get("/key-change", h.keyChangePage)
	r.Post("/key-change", h.keyChangeFormSubmit)
	r.Get("/key-change-success", h.keyChangeSuccessPage)

	// API routes (JSON)
	r.Route("/api/v1", func(r chi.Router) {
		// User registration
		r.Post("/register", h.apiRegister)
		r.Get("/pending", h.apiListPending)
		r.Get("/pending/{username}", h.apiGetPending)
		r.Post("/users/{username}/approve", h.apiApprove)
		r.Post("/users/{username}/reject", h.apiReject)

		// Key change requests
		r.Post("/key-change", h.apiKeyChangeRequest)
		r.Get("/key-changes", h.apiListKeyChanges)
		r.Get("/key-changes/{username}", h.apiGetKeyChange)
		r.Post("/key-changes/{username}/approve", h.apiApproveKeyChange)
		r.Post("/key-changes/{username}/reject", h.apiRejectKeyChange)

		// VM management
		r.Post("/vms", h.apiCreateVM)
		r.Get("/vms", h.apiListVMs)
		r.Get("/vms/{name}", h.apiGetVM)
		r.Delete("/vms/{name}", h.apiDeleteVM)

		// Quota
		r.Get("/quota", h.apiGetQuota)
	})

	// Health check
	r.Get("/health", h.healthCheck)

	return r
}

// JSON response helpers

type apiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Errors  []string    `json:"errors,omitempty"`
}

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, resp apiResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) jsonError(w http.ResponseWriter, status int, message string, errors ...string) {
	h.jsonResponse(w, status, apiResponse{
		Success: false,
		Message: message,
		Errors:  errors,
	})
}

func (h *Handler) jsonSuccess(w http.ResponseWriter, message string, data interface{}) {
	h.jsonResponse(w, http.StatusOK, apiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Health check
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	h.jsonSuccess(w, "OK", nil)
}

// Web pages

func (h *Handler) indexPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/register", http.StatusFound)
}

func (h *Handler) registerPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{}
	if h.config.Recaptcha.Enabled && h.config.Recaptcha.SiteKey != "" {
		data["RecaptchaSiteKey"] = h.config.Recaptcha.SiteKey
	}
	if err := h.templates.ExecuteTemplate(w, "register.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *Handler) successPage(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if err := h.templates.ExecuteTemplate(w, "success.html", map[string]string{
		"Username":       username,
		"BastionAddress": h.config.Bastion.Address,
	}); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *Handler) sshGuidePage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "ssh-guide.html", nil); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *Handler) registerFormSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	input := &model.RegisterInput{
		Username:  r.FormValue("username"),
		Email:     r.FormValue("email"),
		Team:      r.FormValue("team"),
		PublicKey: r.FormValue("public_key"),
	}

	// Helper function to render form with errors
	renderWithErrors := func(errors []string) {
		data := map[string]interface{}{
			"Errors": errors,
			"Input":  input,
		}
		if h.config.Recaptcha.Enabled && h.config.Recaptcha.SiteKey != "" {
			data["RecaptchaSiteKey"] = h.config.Recaptcha.SiteKey
		}
		if err := h.templates.ExecuteTemplate(w, "register.html", data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	}

	// Verify reCAPTCHA if enabled
	if h.config.Recaptcha.Enabled {
		recaptchaResponse := r.FormValue("g-recaptcha-response")
		if recaptchaResponse == "" {
			renderWithErrors([]string{"reCAPTCHA를 완료해주세요"})
			return
		}
		if !h.verifyRecaptcha(recaptchaResponse) {
			renderWithErrors([]string{"reCAPTCHA 검증에 실패했습니다. 다시 시도해주세요"})
			return
		}
	}

	if errors := input.Validate(); len(errors) > 0 {
		renderWithErrors(errors)
		return
	}

	// Validate email domain
	if err := h.validateEmailDomain(input.Email); err != nil {
		renderWithErrors([]string{err.Error()})
		return
	}

	// Create registration request
	req, err := h.createRegistrationRequest(input)
	if err != nil {
		renderWithErrors([]string{err.Error()})
		return
	}

	http.Redirect(w, r, "/success?username="+req.Username, http.StatusFound)
}

// verifyRecaptcha verifies the reCAPTCHA response with Google's API
func (h *Handler) verifyRecaptcha(response string) bool {
	if h.config.Recaptcha.SecretKey == "" {
		log.Printf("Warning: reCAPTCHA secret key is not configured")
		return false
	}

	log.Printf("reCAPTCHA: Verifying response (length=%d)", len(response))

	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify",
		url.Values{
			"secret":   {h.config.Recaptcha.SecretKey},
			"response": {response},
		})
	if err != nil {
		log.Printf("Error verifying reCAPTCHA: %v", err)
		return false
	}
	defer resp.Body.Close()

	var result struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
		Hostname   string   `json:"hostname"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding reCAPTCHA response: %v", err)
		return false
	}

	log.Printf("reCAPTCHA result: success=%v, hostname=%s, errors=%v", result.Success, result.Hostname, result.ErrorCodes)

	return result.Success
}

// validateEmailDomain checks if the email domain is allowed
func (h *Handler) validateEmailDomain(email string) error {
	allowedDomains := h.config.Validation.AllowedEmailDomains
	if len(allowedDomains) == 0 {
		return nil // No restriction
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format")
	}

	domain := strings.ToLower(parts[1])
	for _, allowed := range allowedDomains {
		if strings.ToLower(allowed) == domain {
			return nil
		}
	}

	return fmt.Errorf("이메일 도메인 '%s'는 허용되지 않습니다. 허용된 도메인: %s", domain, strings.Join(allowedDomains, ", "))
}

// API handlers

func (h *Handler) apiRegister(w http.ResponseWriter, r *http.Request) {
	var input model.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.jsonError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	if errors := input.Validate(); len(errors) > 0 {
		h.jsonError(w, http.StatusBadRequest, "Validation failed", errors...)
		return
	}

	// Validate email domain
	if err := h.validateEmailDomain(input.Email); err != nil {
		h.jsonError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	req, err := h.createRegistrationRequest(&input)
	if err != nil {
		h.jsonError(w, http.StatusConflict, err.Error())
		return
	}

	h.jsonSuccess(w, "Registration request submitted", req)
}

func (h *Handler) apiListPending(w http.ResponseWriter, r *http.Request) {
	status := model.StatusPending
	requests, err := h.store.List(&status)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to list requests", err.Error())
		return
	}

	h.jsonSuccess(w, "", requests)
}

func (h *Handler) apiGetPending(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	req, err := h.store.GetByUsername(username)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "Request not found", err.Error())
		return
	}

	h.jsonSuccess(w, "", req)
}

func (h *Handler) apiApprove(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var input model.ApproveInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// Allow empty body, default to "admin"
		input.ProcessedBy = "admin"
	}

	req, err := h.store.GetByUsername(username)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "Request not found", err.Error())
		return
	}

	if req.Status != model.StatusPending {
		h.jsonError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Check if system user already exists
	exists, err := h.provisioner.UserExists(username)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to check user", err.Error())
		return
	}
	if exists {
		h.jsonError(w, http.StatusConflict, "System user already exists")
		return
	}

	// Provision the user
	if err := h.provisioner.CreateUser(req); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to create user", err.Error())
		return
	}

	// Update request status
	req.Status = model.StatusApproved
	req.ProcessedBy = input.ProcessedBy
	req.ProcessedAt = time.Now().Format(time.RFC3339)
	req.UpdatedAt = time.Now()

	if err := h.store.Update(req); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to update request", err.Error())
		return
	}

	h.jsonSuccess(w, "User approved and created", req)
}

func (h *Handler) apiReject(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var input model.RejectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		input.ProcessedBy = "admin"
	}

	req, err := h.store.GetByUsername(username)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "Request not found", err.Error())
		return
	}

	if req.Status != model.StatusPending {
		h.jsonError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Update request status
	req.Status = model.StatusRejected
	req.ProcessedBy = input.ProcessedBy
	req.ProcessedAt = time.Now().Format(time.RFC3339)
	req.RejectReason = input.Reason
	req.UpdatedAt = time.Now()

	if err := h.store.Update(req); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to update request", err.Error())
		return
	}

	h.jsonSuccess(w, "Request rejected", req)
}

// Helper methods

func (h *Handler) createRegistrationRequest(input *model.RegisterInput) (*model.RegistrationRequest, error) {
	// Check if username already has pending request
	exists, err := h.store.ExistsUsername(input.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("사용자명 '%s'으로 이미 등록 요청이 있습니다", input.Username)
	}

	// Check if email already has pending request
	emailExists, err := h.store.ExistsEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, fmt.Errorf("이메일 '%s'으로 이미 등록 요청이 있습니다", input.Email)
	}

	// Check if system user already exists
	userExists, err := h.provisioner.UserExists(input.Username)
	if err != nil {
		return nil, err
	}
	if userExists {
		return nil, fmt.Errorf("사용자명 '%s'은 이미 사용 중입니다", input.Username)
	}

	now := time.Now()
	req := &model.RegistrationRequest{
		ID:        generateID(),
		Username:  input.Username,
		Email:     input.Email,
		Team:      input.Team,
		PublicKey: input.PublicKey,
		Status:    model.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.store.Create(req); err != nil {
		return nil, err
	}

	return req, nil
}

func generateID() string {
	return "req-" + uuid.New().String()[:8]
}

func generateKeyChangeID() string {
	return "keychange-" + uuid.New().String()[:8]
}

// Key change web pages

func (h *Handler) keyChangePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{}
	if h.config.Recaptcha.Enabled && h.config.Recaptcha.SiteKey != "" {
		data["RecaptchaSiteKey"] = h.config.Recaptcha.SiteKey
	}
	if err := h.templates.ExecuteTemplate(w, "key-change.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *Handler) keyChangeSuccessPage(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if err := h.templates.ExecuteTemplate(w, "key-change-success.html", map[string]string{
		"Username": username,
	}); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *Handler) keyChangeFormSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	input := &model.KeyChangeInput{
		Username:     r.FormValue("username"),
		Email:        r.FormValue("email"),
		NewPublicKey: r.FormValue("new_public_key"),
		Reason:       r.FormValue("reason"),
	}

	renderWithErrors := func(errors []string) {
		data := map[string]interface{}{
			"Errors": errors,
			"Input":  input,
		}
		if h.config.Recaptcha.Enabled && h.config.Recaptcha.SiteKey != "" {
			data["RecaptchaSiteKey"] = h.config.Recaptcha.SiteKey
		}
		if err := h.templates.ExecuteTemplate(w, "key-change.html", data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	}

	// Verify reCAPTCHA if enabled
	if h.config.Recaptcha.Enabled {
		recaptchaResponse := r.FormValue("g-recaptcha-response")
		if recaptchaResponse == "" {
			renderWithErrors([]string{"reCAPTCHA를 완료해주세요"})
			return
		}
		if !h.verifyRecaptcha(recaptchaResponse) {
			renderWithErrors([]string{"reCAPTCHA 검증에 실패했습니다. 다시 시도해주세요"})
			return
		}
	}

	if errors := input.Validate(); len(errors) > 0 {
		renderWithErrors(errors)
		return
	}

	// Create key change request
	req, err := h.createKeyChangeRequest(input)
	if err != nil {
		renderWithErrors([]string{err.Error()})
		return
	}

	http.Redirect(w, r, "/key-change-success?username="+req.Username, http.StatusFound)
}

// createKeyChangeRequest creates a key change request after validation
func (h *Handler) createKeyChangeRequest(input *model.KeyChangeInput) (*model.KeyChangeRequest, error) {
	if h.keyChangeStore == nil {
		return nil, fmt.Errorf("키 변경 기능이 비활성화되어 있습니다")
	}

	// Check if user exists
	exists, err := h.provisioner.UserExists(input.Username)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("사용자 '%s'이(가) 존재하지 않습니다", input.Username)
	}

	// Verify email matches the registered email
	registeredEmail, err := h.provisioner.GetUserEmail(input.Username)
	if err != nil {
		// If we can't get the email, allow the request but note in logs
		log.Printf("Warning: could not verify email for user %s: %v", input.Username, err)
	} else if registeredEmail != "" && registeredEmail != input.Email {
		return nil, fmt.Errorf("입력한 이메일이 등록된 이메일과 일치하지 않습니다")
	}

	now := time.Now()
	req := &model.KeyChangeRequest{
		ID:           generateKeyChangeID(),
		Username:     input.Username,
		Email:        input.Email,
		NewPublicKey: input.NewPublicKey,
		Reason:       input.Reason,
		Status:       model.StatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.keyChangeStore.Create(req); err != nil {
		return nil, err
	}

	return req, nil
}

// Key change API handlers

func (h *Handler) apiKeyChangeRequest(w http.ResponseWriter, r *http.Request) {
	var input model.KeyChangeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.jsonError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	if errors := input.Validate(); len(errors) > 0 {
		h.jsonError(w, http.StatusBadRequest, "Validation failed", errors...)
		return
	}

	req, err := h.createKeyChangeRequest(&input)
	if err != nil {
		h.jsonError(w, http.StatusConflict, err.Error())
		return
	}

	h.jsonSuccess(w, "Key change request submitted", req)
}

func (h *Handler) apiListKeyChanges(w http.ResponseWriter, r *http.Request) {
	if h.keyChangeStore == nil {
		h.jsonError(w, http.StatusServiceUnavailable, "Key change feature is disabled")
		return
	}

	status := model.StatusPending
	requests, err := h.keyChangeStore.List(&status)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to list requests", err.Error())
		return
	}

	h.jsonSuccess(w, "", requests)
}

func (h *Handler) apiGetKeyChange(w http.ResponseWriter, r *http.Request) {
	if h.keyChangeStore == nil {
		h.jsonError(w, http.StatusServiceUnavailable, "Key change feature is disabled")
		return
	}

	username := chi.URLParam(r, "username")
	req, err := h.keyChangeStore.GetByUsername(username)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "Request not found", err.Error())
		return
	}

	h.jsonSuccess(w, "", req)
}

func (h *Handler) apiApproveKeyChange(w http.ResponseWriter, r *http.Request) {
	if h.keyChangeStore == nil {
		h.jsonError(w, http.StatusServiceUnavailable, "Key change feature is disabled")
		return
	}

	username := chi.URLParam(r, "username")

	var input model.ApproveInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		input.ProcessedBy = "admin"
	}

	req, err := h.keyChangeStore.GetByUsername(username)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "Request not found", err.Error())
		return
	}

	if req.Status != model.StatusPending {
		h.jsonError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Update the user's SSH key
	if err := h.provisioner.UpdateUserKey(username, req.NewPublicKey); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to update SSH key", err.Error())
		return
	}

	// Update request status
	req.Status = model.StatusApproved
	req.ProcessedBy = input.ProcessedBy
	req.ProcessedAt = time.Now().Format(time.RFC3339)
	req.UpdatedAt = time.Now()

	if err := h.keyChangeStore.Update(req); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to update request", err.Error())
		return
	}

	h.jsonSuccess(w, "Key change approved and applied", req)
}

func (h *Handler) apiRejectKeyChange(w http.ResponseWriter, r *http.Request) {
	if h.keyChangeStore == nil {
		h.jsonError(w, http.StatusServiceUnavailable, "Key change feature is disabled")
		return
	}

	username := chi.URLParam(r, "username")

	var input model.RejectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		input.ProcessedBy = "admin"
	}

	req, err := h.keyChangeStore.GetByUsername(username)
	if err != nil {
		h.jsonError(w, http.StatusNotFound, "Request not found", err.Error())
		return
	}

	if req.Status != model.StatusPending {
		h.jsonError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Update request status
	req.Status = model.StatusRejected
	req.ProcessedBy = input.ProcessedBy
	req.ProcessedAt = time.Now().Format(time.RFC3339)
	req.RejectReason = input.Reason
	req.UpdatedAt = time.Now()

	if err := h.keyChangeStore.Update(req); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "Failed to update request", err.Error())
		return
	}

	h.jsonSuccess(w, "Key change request rejected", req)
}
