package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/opsnexus/tenant-service/internal/domain"
	"github.com/opsnexus/tenant-service/internal/ports"
)

// Handler holds references to all service dependencies.
type Handler struct {
	tenantSvc  ports.TenantService
	orgSvc     ports.OrganizationService
	profileSvc ports.UserProfileService
	logger     *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(
	tenantSvc ports.TenantService,
	orgSvc ports.OrganizationService,
	profileSvc ports.UserProfileService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		tenantSvc:  tenantSvc,
		orgSvc:     orgSvc,
		profileSvc: profileSvc,
		logger:     logger,
	}
}

// ─── Tenant handlers ──────────────────────────────────────────────────────────

// CreateTenant handles POST /api/v1/tenants
func (h *Handler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string            `json:"name"`
		Slug string            `json:"slug"`
		Plan domain.TenantPlan `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenant, err := h.tenantSvc.CreateTenant(r.Context(), ports.CreateTenantRequest{
		Name: body.Name,
		Slug: body.Slug,
		Plan: body.Plan,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, tenant)
}

// ListTenants handles GET /api/v1/tenants
func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	tenants, total, err := h.tenantSvc.ListTenants(r.Context(), page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list tenants")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data":  tenants,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetTenant handles GET /api/v1/tenants/{tenantId}
func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "tenantId")
	tenant, err := h.tenantSvc.GetTenant(r.Context(), id)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, tenant)
}

// UpdateTenant handles PUT /api/v1/tenants/{tenantId}
func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "tenantId")

	var body struct {
		Name   *string              `json:"name"`
		Plan   *domain.TenantPlan   `json:"plan"`
		Status *domain.TenantStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenant, err := h.tenantSvc.UpdateTenant(r.Context(), id, ports.UpdateTenantRequest{
		Name:   body.Name,
		Plan:   body.Plan,
		Status: body.Status,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, tenant)
}

// DeactivateTenant handles DELETE /api/v1/tenants/{tenantId}
func (h *Handler) DeactivateTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "tenantId")
	if err := h.tenantSvc.DeactivateTenant(r.Context(), id); err != nil {
		h.handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

// GetSettings handles GET /api/v1/tenants/{tenantId}/settings
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "tenantId")
	settings, err := h.tenantSvc.GetSettings(r.Context(), id)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

// UpdateSettings handles PUT /api/v1/tenants/{tenantId}/settings
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "tenantId")

	var settings domain.TenantSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.tenantSvc.UpdateSettings(r.Context(), id, settings); err != nil {
		h.handleDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ListMembers handles GET /api/v1/tenants/{tenantId}/members
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	profiles, total, err := h.profileSvc.ListProfiles(r.Context(), tenantID, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list members")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data":  profiles,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// ─── Profile handlers ─────────────────────────────────────────────────────────

// GetProfile handles GET /api/v1/profiles/{userId}
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	profile, err := h.profileSvc.GetProfile(r.Context(), userID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/profiles/{userId}
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")

	var body struct {
		DisplayName    *string `json:"display_name"`
		AvatarURL      *string `json:"avatar_url"`
		Timezone       *string `json:"timezone"`
		Locale         *string `json:"locale"`
		OrganizationID *string `json:"organization_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	profile, err := h.profileSvc.UpdateProfile(r.Context(), userID, ports.UpdateProfileRequest{
		DisplayName:    body.DisplayName,
		AvatarURL:      body.AvatarURL,
		Timezone:       body.Timezone,
		Locale:         body.Locale,
		OrganizationID: body.OrganizationID,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// ─── Organization handlers ────────────────────────────────────────────────────

// ListOrganizations handles GET /api/v1/organizations
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		// Fall back to X-Tenant-ID header (set by TenantContext middleware).
		tenantID, _ = r.Context().Value(TenantIDKey).(string)
	}
	if tenantID == "" {
		respondError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	orgs, err := h.orgSvc.ListOrganizations(r.Context(), tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list organizations")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"data": orgs})
}

// CreateOrganization handles POST /api/v1/organizations
func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID string  `json:"tenant_id"`
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.TenantID == "" {
		body.TenantID, _ = r.Context().Value(TenantIDKey).(string)
	}
	if body.TenantID == "" {
		respondError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	org, err := h.orgSvc.CreateOrganization(r.Context(), body.TenantID, ports.CreateOrgRequest{
		Name:     body.Name,
		Type:     body.Type,
		ParentID: body.ParentID,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, org)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func (h *Handler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrTenantNotFound):
		respondError(w, http.StatusNotFound, "tenant not found")
	case errors.Is(err, domain.ErrTenantAlreadyExists):
		respondError(w, http.StatusConflict, "tenant already exists")
	case errors.Is(err, domain.ErrSlugTaken):
		respondError(w, http.StatusConflict, "slug is already taken")
	case errors.Is(err, domain.ErrOrganizationNotFound):
		respondError(w, http.StatusNotFound, "organization not found")
	case errors.Is(err, domain.ErrProfileNotFound):
		respondError(w, http.StatusNotFound, "profile not found")
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		h.logger.Error("unhandled error", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// Nothing we can do after headers are sent.
		_ = err
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
