package ports

import (
	"context"

	"github.com/opsnexus/document-service/internal/domain"
)

type FormService interface {
	CreateTemplate(ctx context.Context, tenantID, createdBy string, req CreateFormRequest) (*domain.FormTemplate, error)
	GetTemplate(ctx context.Context, id string) (*domain.FormTemplate, error)
	UpdateTemplate(ctx context.Context, id string, req UpdateFormRequest) (*domain.FormTemplate, error)
	PublishTemplate(ctx context.Context, id string) error
	ArchiveTemplate(ctx context.Context, id string) error
	ListTemplates(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormTemplate, int64, error)
	SubmitForm(ctx context.Context, tenantID, userID string, req SubmitFormRequest) (*domain.FormSubmission, error)
	GetSubmission(ctx context.Context, id string) (*domain.FormSubmission, error)
	ListSubmissions(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormSubmission, int64, error)
}

type DocumentService interface {
	UploadDocument(ctx context.Context, tenantID, userID string, req UploadRequest) (*domain.Document, error)
	GetDocument(ctx context.Context, id string) (*domain.Document, error)
	DeleteDocument(ctx context.Context, id string) error
	ListDocuments(ctx context.Context, tenantID string, page, limit int) ([]*domain.Document, int64, error)
	GetVersions(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error)
}

type CreateFormRequest struct {
	Name        string
	Description string
	Fields      []domain.FormField
}

type UpdateFormRequest struct {
	Name        *string
	Description *string
	Fields      []domain.FormField
}

type SubmitFormRequest struct {
	FormID string
	Data   map[string]any
}

type UploadRequest struct {
	Filename  string
	MimeType  string
	SizeBytes int64
	Content   []byte
	CaseID    *string
}
