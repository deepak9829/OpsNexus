package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/opsnexus/document-service/internal/domain"
	"github.com/opsnexus/document-service/internal/ports"
	"go.uber.org/zap"
)

type formService struct {
	formRepo       ports.FormTemplateRepository
	submissionRepo ports.FormSubmissionRepository
	logger         *zap.Logger
}

func NewFormService(
	formRepo ports.FormTemplateRepository,
	submissionRepo ports.FormSubmissionRepository,
	logger *zap.Logger,
) ports.FormService {
	return &formService{
		formRepo:       formRepo,
		submissionRepo: submissionRepo,
		logger:         logger,
	}
}

func (s *formService) CreateTemplate(ctx context.Context, tenantID, createdBy string, req ports.CreateFormRequest) (*domain.FormTemplate, error) {
	now := time.Now().UTC()
	form := &domain.FormTemplate{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Version:     1,
		Fields:      req.Fields,
		Status:      domain.FormStatusDraft,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.formRepo.Create(ctx, form); err != nil {
		s.logger.Error("failed to create form template", zap.Error(err))
		return nil, err
	}

	s.logger.Info("form template created", zap.String("id", form.ID), zap.String("tenant_id", tenantID))
	return form, nil
}

func (s *formService) GetTemplate(ctx context.Context, id string) (*domain.FormTemplate, error) {
	return s.formRepo.FindByID(ctx, id)
}

func (s *formService) UpdateTemplate(ctx context.Context, id string, req ports.UpdateFormRequest) (*domain.FormTemplate, error) {
	form, err := s.formRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		form.Name = *req.Name
	}
	if req.Description != nil {
		form.Description = *req.Description
	}
	if req.Fields != nil {
		form.Fields = req.Fields
		form.Version++
	}
	form.UpdatedAt = time.Now().UTC()

	if err := s.formRepo.Update(ctx, form); err != nil {
		return nil, err
	}
	return form, nil
}

func (s *formService) PublishTemplate(ctx context.Context, id string) error {
	return s.formRepo.UpdateStatus(ctx, id, domain.FormStatusPublished)
}

func (s *formService) ArchiveTemplate(ctx context.Context, id string) error {
	return s.formRepo.UpdateStatus(ctx, id, domain.FormStatusArchived)
}

func (s *formService) ListTemplates(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormTemplate, int64, error) {
	return s.formRepo.ListByTenant(ctx, tenantID, page, limit)
}

func (s *formService) SubmitForm(ctx context.Context, tenantID, userID string, req ports.SubmitFormRequest) (*domain.FormSubmission, error) {
	form, err := s.formRepo.FindByID(ctx, req.FormID)
	if err != nil {
		return nil, err
	}

	for _, field := range form.Fields {
		if field.Required {
			val, exists := req.Data[field.Name]
			if !exists || val == nil || val == "" {
				return nil, fmt.Errorf("%w: field '%s' is required", domain.ErrInvalidFormData, field.Name)
			}
		}
	}

	now := time.Now().UTC()
	submission := &domain.FormSubmission{
		ID:          uuid.New().String(),
		FormID:      req.FormID,
		TenantID:    tenantID,
		SubmittedBy: userID,
		Data:        req.Data,
		Status:      domain.SubmissionStatusSubmitted,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.submissionRepo.Create(ctx, submission); err != nil {
		s.logger.Error("failed to create submission", zap.Error(err))
		return nil, err
	}

	s.logger.Info("form submitted", zap.String("id", submission.ID), zap.String("form_id", req.FormID))
	return submission, nil
}

func (s *formService) GetSubmission(ctx context.Context, id string) (*domain.FormSubmission, error) {
	return s.submissionRepo.FindByID(ctx, id)
}

func (s *formService) ListSubmissions(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormSubmission, int64, error) {
	return s.submissionRepo.ListByTenant(ctx, tenantID, page, limit)
}
