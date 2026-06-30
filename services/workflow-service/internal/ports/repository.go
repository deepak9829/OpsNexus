package ports

import (
	"context"

	"github.com/opsnexus/workflow-service/internal/domain"
)

type CaseFilter struct {
	Status     *domain.CaseStatus
	Priority   *domain.CasePriority
	AssigneeID *string
	ReporterID *string
	Tags       []string
}

type CaseRepository interface {
	Create(ctx context.Context, c *domain.Case) error
	FindByID(ctx context.Context, id string) (*domain.Case, error)
	FindByNumber(ctx context.Context, tenantID, caseNumber string) (*domain.Case, error)
	Update(ctx context.Context, c *domain.Case) error
	List(ctx context.Context, tenantID string, filter CaseFilter, page, limit int) ([]*domain.Case, int64, error)
	RecordTransition(ctx context.Context, t *domain.CaseTransition) error
	GetTransitionHistory(ctx context.Context, caseID string) ([]*domain.CaseTransition, error)
	NextCaseNumber(ctx context.Context, tenantID string) (string, error)
}

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	FindByID(ctx context.Context, id string) (*domain.Task, error)
	ListByCase(ctx context.Context, caseID string) ([]*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id string) error
}

type WorkflowTemplateRepository interface {
	Create(ctx context.Context, w *domain.WorkflowTemplate) error
	FindByID(ctx context.Context, id string) (*domain.WorkflowTemplate, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.WorkflowTemplate, error)
	Update(ctx context.Context, w *domain.WorkflowTemplate) error
}

type CommentRepository interface {
	Create(ctx context.Context, c *domain.Comment) error
	FindByID(ctx context.Context, id string) (*domain.Comment, error)
	ListByCase(ctx context.Context, caseID string) ([]*domain.Comment, error)
	Delete(ctx context.Context, id string) error
}
