package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/opsnexus/workflow-service/internal/domain"
	"github.com/opsnexus/workflow-service/internal/ports"
	"go.uber.org/zap"
)

type Handler struct {
	caseService     ports.CaseService
	taskService     ports.TaskService
	workflowService ports.WorkflowTemplateService
	logger          *zap.Logger
}

func NewHandler(
	caseService ports.CaseService,
	taskService ports.TaskService,
	workflowService ports.WorkflowTemplateService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		caseService:     caseService,
		taskService:     taskService,
		workflowService: workflowService,
		logger:          logger,
	}
}

// ---- Cases ----

func (h *Handler) CreateCase(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r.Context())
	userID := getUserID(r.Context())

	var body struct {
		Title       string              `json:"title"`
		Description string              `json:"description"`
		Priority    domain.CasePriority `json:"priority"`
		WorkflowID  *string             `json:"workflowId"`
		Tags        []string            `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req := ports.CreateCaseRequest{
		Title:       body.Title,
		Description: body.Description,
		Priority:    body.Priority,
		WorkflowID:  body.WorkflowID,
		Tags:        body.Tags,
	}

	c, err := h.caseService.CreateCase(r.Context(), tenantID, userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) ListCases(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r.Context())

	filter := ports.CaseFilter{}
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.CaseStatus(s)
		filter.Status = &st
	}
	if p := r.URL.Query().Get("priority"); p != "" {
		pr := domain.CasePriority(p)
		filter.Priority = &pr
	}
	if a := r.URL.Query().Get("assignee"); a != "" {
		filter.AssigneeID = &a
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 20
	}

	cases, total, err := h.caseService.ListCases(r.Context(), tenantID, filter, page, limit)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  cases,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) GetCase(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	c, err := h.caseService.GetCase(r.Context(), caseID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) UpdateCase(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")

	var body struct {
		Title       *string              `json:"title"`
		Description *string              `json:"description"`
		Priority    *domain.CasePriority `json:"priority"`
		Tags        []string             `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req := ports.UpdateCaseRequest{
		Title:       body.Title,
		Description: body.Description,
		Priority:    body.Priority,
		Tags:        body.Tags,
	}

	c, err := h.caseService.UpdateCase(r.Context(), caseID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) TransitionCase(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	userID := getUserID(r.Context())

	var body struct {
		ToStatus domain.CaseStatus `json:"toStatus"`
		Reason   string            `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req := ports.TransitionRequest{
		ToStatus:    body.ToStatus,
		Reason:      body.Reason,
		PerformedBy: userID,
	}

	c, err := h.caseService.TransitionCase(r.Context(), caseID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) AssignCase(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")

	var body struct {
		AssigneeID string `json:"assigneeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if body.AssigneeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "assigneeId is required"})
		return
	}

	c, err := h.caseService.AssignCase(r.Context(), caseID, body.AssigneeID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ---- Tasks ----

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	tasks, err := h.taskService.ListByCase(r.Context(), caseID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")

	var body struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		AssigneeID  *string `json:"assigneeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req := ports.CreateTaskRequest{
		Title:       body.Title,
		Description: body.Description,
		AssigneeID:  body.AssigneeID,
	}

	task, err := h.taskService.CreateTask(r.Context(), caseID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskId")

	var body struct {
		Title       *string            `json:"title"`
		Description *string            `json:"description"`
		Status      *domain.TaskStatus `json:"status"`
		AssigneeID  *string            `json:"assigneeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req := ports.UpdateTaskRequest{
		Title:       body.Title,
		Description: body.Description,
		Status:      body.Status,
		AssigneeID:  body.AssigneeID,
	}

	task, err := h.taskService.UpdateTask(r.Context(), taskID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskId")
	task, err := h.taskService.CompleteTask(r.Context(), taskID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// ---- Comments ----

func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	comments, err := h.caseService.ListComments(r.Context(), caseID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	userID := getUserID(r.Context())

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	comment, err := h.caseService.AddComment(r.Context(), caseID, userID, body.Body)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

// ---- Workflows ----

func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r.Context())
	templates, err := h.workflowService.ListTemplates(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r.Context())

	var body struct {
		Name            string              `json:"name"`
		Description     string              `json:"description"`
		States          []string            `json:"states"`
		Transitions     []domain.Transition `json:"transitions"`
		DefaultPriority domain.CasePriority `json:"defaultPriority"`
		SLAHours        int                 `json:"slaHours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req := ports.CreateWorkflowTemplateRequest{
		Name:            body.Name,
		Description:     body.Description,
		States:          body.States,
		Transitions:     body.Transitions,
		DefaultPriority: body.DefaultPriority,
		SLAHours:        body.SLAHours,
	}

	wf, err := h.workflowService.CreateTemplate(r.Context(), tenantID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, wf)
}

// ---- Helpers ----

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCaseNotFound),
		errors.Is(err, domain.ErrTaskNotFound),
		errors.Is(err, domain.ErrWorkflowNotFound),
		errors.Is(err, domain.ErrCommentNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidTransition):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrPermissionDenied):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	default:
		h.logger.Error("internal error", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}
