package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/opsnexus/notification-service/internal/domain"
	"github.com/opsnexus/notification-service/internal/ports"
	"go.uber.org/zap"
)

type Handler struct {
	notifSvc ports.NotificationService
	auditSvc ports.AuditService
	logger   *zap.Logger
}

func NewHandler(notifSvc ports.NotificationService, auditSvc ports.AuditService, logger *zap.Logger) *Handler {
	return &Handler{notifSvc: notifSvc, auditSvc: auditSvc, logger: logger}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func getPagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}

func getTenantID(r *http.Request) string {
	t := r.Header.Get("X-Tenant-ID")
	if t == "" {
		t = "default"
	}
	return t
}

func getUserID(r *http.Request) string {
	u := r.Header.Get("X-User-ID")
	if u == "" {
		u = "anonymous"
	}
	return u
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "notification-service"})
}

// Notification handlers

func (h *Handler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var req ports.SendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TenantID == "" {
		req.TenantID = getTenantID(r)
	}

	n, err := h.notifSvc.Send(r.Context(), req)
	if err != nil {
		h.logger.Error("send notification error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

func (h *Handler) SendBulkNotifications(w http.ResponseWriter, r *http.Request) {
	var reqs []ports.SendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.notifSvc.SendBulk(r.Context(), reqs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "sent"})
}

func (h *Handler) GetNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.notifSvc.GetNotification(r.Context(), getTenantID(r), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotificationNotFound) {
			writeError(w, http.StatusNotFound, "notification not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = getUserID(r)
	}
	page, limit := getPagination(r)

	notifications, total, err := h.notifSvc.ListForUser(r.Context(), tenantID, userID, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": notifications, "total": total, "page": page, "limit": limit})
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.notifSvc.MarkRead(r.Context(), getTenantID(r), id); err != nil {
		if errors.Is(err, domain.ErrNotificationNotFound) {
			writeError(w, http.StatusNotFound, "notification not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "read"})
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	userID := getUserID(r)
	if err := h.notifSvc.MarkAllRead(r.Context(), tenantID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "all read"})
}

func (h *Handler) DismissNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.notifSvc.Dismiss(r.Context(), getTenantID(r), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

// Audit handlers

func (h *Handler) RecordAuditEvent(w http.ResponseWriter, r *http.Request) {
	var req ports.RecordAuditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TenantID == "" {
		req.TenantID = getTenantID(r)
	}

	event, err := h.auditSvc.Record(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h *Handler) GetAuditEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := h.auditSvc.GetEvent(r.Context(), getTenantID(r), id)
	if err != nil {
		if errors.Is(err, domain.ErrAuditEventNotFound) {
			writeError(w, http.StatusNotFound, "audit event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h *Handler) QueryAuditEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	page, limit := getPagination(r)

	filter := ports.AuditFilter{}
	if v := r.URL.Query().Get("actor_id"); v != "" {
		filter.ActorID = &v
	}
	if v := r.URL.Query().Get("action"); v != "" {
		filter.Action = &v
	}
	if v := r.URL.Query().Get("resource"); v != "" {
		filter.Resource = &v
	}
	if v := r.URL.Query().Get("resource_id"); v != "" {
		filter.ResourceID = &v
	}

	events, total, err := h.auditSvc.QueryEvents(r.Context(), tenantID, filter, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": events, "total": total, "page": page, "limit": limit})
}
