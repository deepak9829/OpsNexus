package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opsnexus/auth-service/internal/domain"
	"github.com/opsnexus/auth-service/internal/ports"
	"go.uber.org/zap"
)

type Handler struct {
	authSvc ports.AuthService
	logger  *zap.Logger
}

func NewHandler(authSvc ports.AuthService, logger *zap.Logger) *Handler {
	return &Handler{
		authSvc: authSvc,
		logger:  logger,
	}
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func mapDomainError(err error) (int, string, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error()
	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, "USER_NOT_FOUND", err.Error()
	case errors.Is(err, domain.ErrUserAlreadyExists):
		return http.StatusConflict, "USER_ALREADY_EXISTS", err.Error()
	case errors.Is(err, domain.ErrPermissionDenied):
		return http.StatusForbidden, "PERMISSION_DENIED", err.Error()
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, "INVALID_INPUT", err.Error()
	case errors.Is(err, domain.ErrUserInactive):
		return http.StatusForbidden, "USER_INACTIVE", err.Error()
	case errors.Is(err, domain.ErrTokenInvalid):
		return http.StatusUnauthorized, "TOKEN_INVALID", err.Error()
	case errors.Is(err, domain.ErrTokenExpired):
		return http.StatusUnauthorized, "TOKEN_EXPIRED", err.Error()
	case errors.Is(err, domain.ErrTokenRevoked):
		return http.StatusUnauthorized, "TOKEN_REVOKED", err.Error()
	case errors.Is(err, domain.ErrRoleNotFound):
		return http.StatusNotFound, "ROLE_NOT_FOUND", err.Error()
	case errors.Is(err, domain.ErrTenantMismatch):
		return http.StatusBadRequest, "TENANT_MISMATCH", err.Error()
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred"
	}
}

type registerRequest struct {
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type assignRoleRequest struct {
	RoleID string `json:"role_id"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	user, err := h.authSvc.Register(r.Context(), ports.RegisterRequest{
		TenantID:  req.TenantID,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		status, code, msg := mapDomainError(err)
		respondError(w, status, code, msg)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"id":         user.ID,
			"tenant_id":  user.TenantID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"status":     user.Status,
			"created_at": user.CreatedAt,
		},
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	tokenPair, err := h.authSvc.Login(r.Context(), ports.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		status, code, msg := mapDomainError(err)
		respondError(w, status, code, msg)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_in":    tokenPair.ExpiresIn,
			"token_type":    tokenPair.TokenType,
		},
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	tokenPair, err := h.authSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		status, code, msg := mapDomainError(err)
		respondError(w, status, code, msg)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_in":    tokenPair.ExpiresIn,
			"token_type":    tokenPair.TokenType,
		},
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.authSvc.Logout(r.Context(), req.RefreshToken); err != nil {
		status, code, msg := mapDomainError(err)
		respondError(w, status, code, msg)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{
			"message": "logged out successfully",
		},
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ContextKeyUserID).(string)
	if !ok || userID == "" {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	user, err := h.authSvc.GetCurrentUser(r.Context(), userID)
	if err != nil {
		status, code, msg := mapDomainError(err)
		respondError(w, status, code, msg)
		return
	}

	roles := make([]map[string]any, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, map[string]any{
			"id":   role.ID,
			"name": role.Name,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":         user.ID,
			"tenant_id":  user.TenantID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"status":     user.Status,
			"roles":      roles,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ContextKeyUserID).(string)
	if !ok || userID == "" {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var body struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	user, err := h.authSvc.GetCurrentUser(r.Context(), userID)
	if err != nil {
		status, code, msg := mapDomainError(err)
		respondError(w, status, code, msg)
		return
	}

	if body.FirstName != "" {
		user.FirstName = body.FirstName
	}
	if body.LastName != "" {
		user.LastName = body.LastName
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"status":     user.Status,
		},
	})
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"data": []map[string]any{},
	})
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]string{
			"message": "role creation endpoint - wire roleRepo to implement fully",
		},
	})
}

func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "user ID is required")
		return
	}

	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.authSvc.AssignRole(r.Context(), userID, req.RoleID); err != nil {
		status, code, msg := mapDomainError(err)
		respondError(w, status, code, msg)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{
			"message": "role assigned successfully",
		},
	})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(ContextKeyTenantID).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "tenant ID not found")
		return
	}

	page := 1
	limit := 20
	if v := r.URL.Query().Get("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}

	users, total, err := h.authSvc.ListUsers(r.Context(), tenantID, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users")
		return
	}

	type roleItem struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type userItem struct {
		ID        string     `json:"id"`
		TenantID  string     `json:"tenant_id"`
		Email     string     `json:"email"`
		FirstName string     `json:"first_name"`
		LastName  string     `json:"last_name"`
		Status    string     `json:"status"`
		Roles     []roleItem `json:"roles"`
		CreatedAt string     `json:"created_at"`
		UpdatedAt string     `json:"updated_at"`
	}

	items := make([]userItem, 0, len(users))
	for _, u := range users {
		roles := make([]roleItem, 0, len(u.Roles))
		for _, r := range u.Roles {
			roles = append(roles, roleItem{ID: r.ID, Name: r.Name})
		}
		items = append(items, userItem{
			ID:        u.ID,
			TenantID:  u.TenantID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Status:    string(u.Status),
			Roles:     roles,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data":  items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "user ID is required")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.authSvc.UpdateUserStatus(r.Context(), userID, body.Status); err != nil {
		status, code, msg := mapDomainError(err)
		respondError(w, status, code, msg)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"message": "user updated"}})
}
