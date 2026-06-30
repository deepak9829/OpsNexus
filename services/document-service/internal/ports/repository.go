package ports

import (
	"context"

	"github.com/opsnexus/document-service/internal/domain"
)

type FormTemplateRepository interface {
	Create(ctx context.Context, form *domain.FormTemplate) error
	FindByID(ctx context.Context, id string) (*domain.FormTemplate, error)
	ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormTemplate, int64, error)
	Update(ctx context.Context, form *domain.FormTemplate) error
	UpdateStatus(ctx context.Context, id string, status domain.FormStatus) error
}

type FormSubmissionRepository interface {
	Create(ctx context.Context, sub *domain.FormSubmission) error
	FindByID(ctx context.Context, id string) (*domain.FormSubmission, error)
	ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormSubmission, int64, error)
	ListByForm(ctx context.Context, formID string, page, limit int) ([]*domain.FormSubmission, int64, error)
	UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus) error
}

type DocumentRepository interface {
	Create(ctx context.Context, doc *domain.Document) error
	FindByID(ctx context.Context, id string) (*domain.Document, error)
	ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.Document, int64, error)
	Delete(ctx context.Context, id string) error
	AddVersion(ctx context.Context, version *domain.DocumentVersion) error
	ListVersions(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error)
}
