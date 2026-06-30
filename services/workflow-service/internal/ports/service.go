package ports

import (
	"context"
	"time"

	"github.com/opsnexus/workflow-service/internal/domain"
)

type CreateCaseRequest struct {
	Title       string
	Description string
	Priority    domain.CasePriority
	WorkflowID  *string
	Tags        []string
}

type UpdateCaseRequest struct {
	Title       *string
	Description *string
	Priority    *domain.CasePriority
	Tags        []string
}

type TransitionRequest struct {
	ToStatus    domain.CaseStatus
	Reason      string
	PerformedBy string
}

type CreateTaskRequest struct {
	Title       string
	Description string
	AssigneeID  *string
	DueAt       *time.Time
}

type UpdateTaskRequest struct {
	Title       *string
	Description *string
	Status      *domain.TaskStatus
	AssigneeID  *string
	DueAt       *time.Time
}

type CaseService interface {
	CreateCase(ctx context.Context, tenantID, reporterID string, req CreateCaseRequest) (*domain.Case, error)
	GetCase(ctx context.Context, id string) (*domain.Case, error)
	UpdateCase(ctx context.Context, id string, req UpdateCaseRequest) (*domain.Case, error)
	TransitionCase(ctx context.Context, id string, req TransitionRequest) (*domain.Case, error)
	AssignCase(ctx context.Context, id, assigneeID string) (*domain.Case, error)
	ListCases(ctx context.Context, tenantID string, filter CaseFilter, page, limit int) ([]*domain.Case, int64, error)
	AddComment(ctx context.Context, caseID, authorID string, body string) (*domain.Comment, error)
	ListComments(ctx context.Context, caseID string) ([]*domain.Comment, error)
}

type TaskService interface {
	CreateTask(ctx context.Context, caseID string, req CreateTaskRequest) (*domain.Task, error)
	GetTask(ctx context.Context, id string) (*domain.Task, error)
	UpdateTask(ctx context.Context, id string, req UpdateTaskRequest) (*domain.Task, error)
	CompleteTask(ctx context.Context, id string) (*domain.Task, error)
	ListByCase(ctx context.Context, caseID string) ([]*domain.Task, error)
}

type WorkflowTemplateService interface {
	CreateTemplate(ctx context.Context, tenantID string, req CreateWorkflowTemplateRequest) (*domain.WorkflowTemplate, error)
	GetTemplate(ctx context.Context, id string) (*domain.WorkflowTemplate, error)
	ListTemplates(ctx context.Context, tenantID string) ([]*domain.WorkflowTemplate, error)
}

type CreateWorkflowTemplateRequest struct {
	Name            string
	Description     string
	States          []string
	Transitions     []domain.Transition
	DefaultPriority domain.CasePriority
	SLAHours        int
}
