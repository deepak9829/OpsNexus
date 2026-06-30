package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/opsnexus/document-service/internal/domain"
	"github.com/opsnexus/document-service/internal/ports"
	"go.uber.org/zap"
)

const maxUploadSize = 50 * 1024 * 1024 // 50MB

type Handler struct {
	formService     ports.FormService
	documentService ports.DocumentService
	logger          *zap.Logger
}

func NewHandler(formSvc ports.FormService, docSvc ports.DocumentService, logger *zap.Logger) *Handler {
	return &Handler{
		formService:     formSvc,
		documentService: docSvc,
		logger:          logger,
	}
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "document-service"})
}

func (h *Handler) CreateFormTemplate(w http.ResponseWriter, r *http.Request) {
	var req ports.CreateFormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	form, err := h.formService.CreateTemplate(r.Context(), getTenantID(r), getUserID(r), req)
	if err != nil {
		h.logger.Error("create template error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, form)
}

func (h *Handler) GetFormTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	form, err := h.formService.GetTemplate(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrFormNotFound) {
			writeError(w, http.StatusNotFound, "form template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, form)
}

func (h *Handler) UpdateFormTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req ports.UpdateFormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	form, err := h.formService.UpdateTemplate(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, domain.ErrFormNotFound) {
			writeError(w, http.StatusNotFound, "form template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, form)
}

func (h *Handler) PublishFormTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.formService.PublishTemplate(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrFormNotFound) {
			writeError(w, http.StatusNotFound, "form template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func (h *Handler) ArchiveFormTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.formService.ArchiveTemplate(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrFormNotFound) {
			writeError(w, http.StatusNotFound, "form template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (h *Handler) ListFormTemplates(w http.ResponseWriter, r *http.Request) {
	page, limit := getPagination(r)
	forms, total, err := h.formService.ListTemplates(r.Context(), getTenantID(r), page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": forms, "total": total, "page": page, "limit": limit})
}

func (h *Handler) SubmitForm(w http.ResponseWriter, r *http.Request) {
	var req ports.SubmitFormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sub, err := h.formService.SubmitForm(r.Context(), getTenantID(r), getUserID(r), req)
	if err != nil {
		if errors.Is(err, domain.ErrFormNotFound) {
			writeError(w, http.StatusNotFound, "form template not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidFormData) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (h *Handler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sub, err := h.formService.GetSubmission(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSubmissionNotFound) {
			writeError(w, http.StatusNotFound, "submission not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	page, limit := getPagination(r)
	subs, total, err := h.formService.ListSubmissions(r.Context(), getTenantID(r), page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": subs, "total": total, "page": page, "limit": limit})
}

func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var caseID *string
	if cid := r.FormValue("case_id"); cid != "" {
		caseID = &cid
	}

	req := ports.UploadRequest{
		Filename:  header.Filename,
		MimeType:  mimeType,
		SizeBytes: header.Size,
		Content:   content,
		CaseID:    caseID,
	}

	doc, err := h.documentService.UploadDocument(r.Context(), getTenantID(r), getUserID(r), req)
	if err != nil {
		if errors.Is(err, domain.ErrFileTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		if errors.Is(err, domain.ErrUnsupportedFileType) {
			writeError(w, http.StatusUnsupportedMediaType, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	doc, err := h.documentService.GetDocument(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.documentService.DeleteDocument(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	page, limit := getPagination(r)
	docs, total, err := h.documentService.ListDocuments(r.Context(), getTenantID(r), page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": docs, "total": total, "page": page, "limit": limit})
}

func (h *Handler) GetDocumentVersions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	versions, err := h.documentService.GetVersions(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": versions})
}
