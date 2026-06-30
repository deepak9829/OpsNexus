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

type caseService struct {
	caseRepo     ports.CaseRepository
	workflowRepo ports.WorkflowTemplateRepository
	commentRepo  ports.CommentRepository
	logger       *zap.Logger
}

func NewCaseService(
	caseRepo ports.CaseRepository,
	workflowRepo ports.WorkflowTemplateRepository,
	commentRepo ports.CommentRepository,
	logger *zap.Logger,
) ports.CaseService {
	return &caseService{
		caseRepo:     caseRepo,
		workflowRepo: workflowRepo,
		commentRepo:  commentRepo,
		logger:       logger,
	}
}

func (s *caseService) CreateCase(ctx context.Context, tenantID, reporterID string, req ports.CreateCaseRequest) (*domain.Case, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenantID is required", domain.ErrInvalidInput)
	}
	if reporterID == "" {
		return nil, fmt.Errorf("%w: reporterID is required", domain.ErrInvalidInput)
	}

	caseNumber, err := s.caseRepo.NextCaseNumber(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("generating case number: %w", err)
	}

	now := time.Now().UTC()
	c := &domain.Case{
		ID:          uuid.NewString(),
		TenantID:    tenantID,
		CaseNumber:  caseNumber,
		Title:       req.Title,
		Description: req.Description,
		Status:      domain.CaseStatusNew,
		Priority:    req.Priority,
		ReporterID:  reporterID,
		WorkflowID:  req.WorkflowID,
		Tags:        req.Tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if c.Priority == "" {
		c.Priority = domain.PriorityMedium
	}

	// Resolve SLA from workflow template
	if req.WorkflowID != nil && *req.WorkflowID != "" {
		wf, err := s.workflowRepo.FindByID(ctx, *req.WorkflowID)
		if err == nil && wf.SLAHours > 0 {
			dueAt := now.Add(time.Duration(wf.SLAHours) * time.Hour)
			c.SLA = domain.SLA{
				DueAt:    &dueAt,
				Breached: false,
			}
			if c.Priority == "" {
				c.Priority = wf.DefaultPriority
			}
		}
	}

	if err := s.caseRepo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("creating case: %w", err)
	}

	s.logger.Info("case created", zap.String("caseID", c.ID), zap.String("caseNumber", c.CaseNumber))
	return c, nil
}

func (s *caseService) GetCase(ctx context.Context, id string) (*domain.Case, error) {
	c, err := s.caseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Check SLA breach
	if c.SLA.DueAt != nil && !c.SLA.Breached && time.Now().UTC().After(*c.SLA.DueAt) {
		c.SLA.Breached = true
		_ = s.caseRepo.Update(ctx, c)
	}
	return c, nil
}

func (s *caseService) UpdateCase(ctx context.Context, id string, req ports.UpdateCaseRequest) (*domain.Case, error) {
	c, err := s.caseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, fmt.Errorf("%w: title cannot be empty", domain.ErrInvalidInput)
		}
		c.Title = *req.Title
	}
	if req.Description != nil {
		c.Description = *req.Description
	}
	if req.Priority != nil {
		c.Priority = *req.Priority
	}
	if req.Tags != nil {
		c.Tags = req.Tags
	}

	c.UpdatedAt = time.Now().UTC()

	if err := s.caseRepo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("updating case: %w", err)
	}
	return c, nil
}

func (s *caseService) TransitionCase(ctx context.Context, id string, req ports.TransitionRequest) (*domain.Case, error) {
	c, err := s.caseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if c.Status == req.ToStatus {
		return c, nil
	}

	// Validate transition against workflow template if attached
	if c.WorkflowID != nil && *c.WorkflowID != "" {
		wf, err := s.workflowRepo.FindByID(ctx, *c.WorkflowID)
		if err == nil {
			if !isTransitionAllowed(wf.Transitions, c.Status, req.ToStatus) {
				return nil, fmt.Errorf("%w: cannot transition from %s to %s", domain.ErrInvalidTransition, c.Status, req.ToStatus)
			}
		}
	}

	now := time.Now().UTC()
	fromStatus := c.Status
	c.Status = req.ToStatus
	c.UpdatedAt = now

	if req.ToStatus == domain.CaseStatusResolved {
		c.ResolvedAt = &now
	}
	if req.ToStatus == domain.CaseStatusClosed {
		c.ClosedAt = &now
	}

	// Check SLA breach on transition
	if c.SLA.DueAt != nil && !c.SLA.Breached && now.After(*c.SLA.DueAt) {
		c.SLA.Breached = true
	}

	if err := s.caseRepo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("updating case status: %w", err)
	}

	transition := &domain.CaseTransition{
		ID:          uuid.NewString(),
		CaseID:      c.ID,
		FromStatus:  fromStatus,
		ToStatus:    req.ToStatus,
		Reason:      req.Reason,
		PerformedBy: req.PerformedBy,
		PerformedAt: now,
	}
	if err := s.caseRepo.RecordTransition(ctx, transition); err != nil {
		s.logger.Error("recording transition failed", zap.Error(err))
	}

	s.logger.Info("case transitioned",
		zap.String("caseID", c.ID),
		zap.String("from", string(fromStatus)),
		zap.String("to", string(req.ToStatus)),
	)
	return c, nil
}

func (s *caseService) AssignCase(ctx context.Context, id, assigneeID string) (*domain.Case, error) {
	c, err := s.caseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c.AssigneeID = &assigneeID
	c.UpdatedAt = time.Now().UTC()
	if err := s.caseRepo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("assigning case: %w", err)
	}
	return c, nil
}

func (s *caseService) ListCases(ctx context.Context, tenantID string, filter ports.CaseFilter, page, limit int) ([]*domain.Case, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.caseRepo.List(ctx, tenantID, filter, page, limit)
}

func (s *caseService) AddComment(ctx context.Context, caseID, authorID string, body string) (*domain.Comment, error) {
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("%w: comment body is required", domain.ErrInvalidInput)
	}
	// verify case exists and retrieve tenantID
	c, err := s.caseRepo.FindByID(ctx, caseID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	comment := &domain.Comment{
		ID:        uuid.NewString(),
		CaseID:    caseID,
		TenantID:  c.TenantID,
		AuthorID:  authorID,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("creating comment: %w", err)
	}
	return comment, nil
}

func (s *caseService) ListComments(ctx context.Context, caseID string) ([]*domain.Comment, error) {
	return s.commentRepo.ListByCase(ctx, caseID)
}

func isTransitionAllowed(transitions []domain.Transition, from, to domain.CaseStatus) bool {
	for _, t := range transitions {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}
