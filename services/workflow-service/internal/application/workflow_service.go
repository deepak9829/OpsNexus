package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opsnexus/workflow-service/internal/domain"
	"github.com/opsnexus/workflow-service/internal/ports"
	"go.uber.org/zap"
)

type workflowService struct {
	wfRepo ports.WorkflowTemplateRepository
	logger *zap.Logger
}

func NewWorkflowService(wfRepo ports.WorkflowTemplateRepository, logger *zap.Logger) ports.WorkflowTemplateService {
	return &workflowService{
		wfRepo: wfRepo,
		logger: logger,
	}
}

func (s *workflowService) CreateTemplate(ctx context.Context, tenantID string, req ports.CreateWorkflowTemplateRequest) (*domain.WorkflowTemplate, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenantID is required", domain.ErrInvalidInput)
	}

	now := time.Now().UTC()
	wf := &domain.WorkflowTemplate{
		ID:              uuid.NewString(),
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		States:          req.States,
		Transitions:     req.Transitions,
		DefaultPriority: req.DefaultPriority,
		SLAHours:        req.SLAHours,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if wf.DefaultPriority == "" {
		wf.DefaultPriority = domain.PriorityMedium
	}

	if err := s.wfRepo.Create(ctx, wf); err != nil {
		return nil, fmt.Errorf("creating workflow template: %w", err)
	}

	s.logger.Info("workflow template created", zap.String("workflowID", wf.ID), zap.String("name", wf.Name))
	return wf, nil
}

func (s *workflowService) GetTemplate(ctx context.Context, id string) (*domain.WorkflowTemplate, error) {
	return s.wfRepo.FindByID(ctx, id)
}

func (s *workflowService) ListTemplates(ctx context.Context, tenantID string) ([]*domain.WorkflowTemplate, error) {
	return s.wfRepo.ListByTenant(ctx, tenantID)
}
