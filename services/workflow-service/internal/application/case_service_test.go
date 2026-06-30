package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/opsnexus/workflow-service/internal/application"
	"github.com/opsnexus/workflow-service/internal/domain"
	"github.com/opsnexus/workflow-service/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Mock implementations
type mockCaseRepo struct{ mock.Mock }

func (m *mockCaseRepo) Create(ctx context.Context, c *domain.Case) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
func (m *mockCaseRepo) FindByID(ctx context.Context, id string) (*domain.Case, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Case), args.Error(1)
}
func (m *mockCaseRepo) FindByNumber(ctx context.Context, tenantID, caseNumber string) (*domain.Case, error) {
	args := m.Called(ctx, tenantID, caseNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Case), args.Error(1)
}
func (m *mockCaseRepo) Update(ctx context.Context, c *domain.Case) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
func (m *mockCaseRepo) List(ctx context.Context, tenantID string, filter ports.CaseFilter, page, limit int) ([]*domain.Case, int64, error) {
	args := m.Called(ctx, tenantID, filter, page, limit)
	return args.Get(0).([]*domain.Case), args.Get(1).(int64), args.Error(2)
}
func (m *mockCaseRepo) RecordTransition(ctx context.Context, t *domain.CaseTransition) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}
func (m *mockCaseRepo) GetTransitionHistory(ctx context.Context, caseID string) ([]*domain.CaseTransition, error) {
	args := m.Called(ctx, caseID)
	return args.Get(0).([]*domain.CaseTransition), args.Error(1)
}
func (m *mockCaseRepo) NextCaseNumber(ctx context.Context, tenantID string) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

type mockWorkflowRepo struct{ mock.Mock }

func (m *mockWorkflowRepo) Create(ctx context.Context, w *domain.WorkflowTemplate) error {
	args := m.Called(ctx, w)
	return args.Error(0)
}
func (m *mockWorkflowRepo) FindByID(ctx context.Context, id string) (*domain.WorkflowTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkflowTemplate), args.Error(1)
}
func (m *mockWorkflowRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.WorkflowTemplate, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*domain.WorkflowTemplate), args.Error(1)
}
func (m *mockWorkflowRepo) Update(ctx context.Context, w *domain.WorkflowTemplate) error {
	args := m.Called(ctx, w)
	return args.Error(0)
}

type mockCommentRepo struct{ mock.Mock }

func (m *mockCommentRepo) Create(ctx context.Context, c *domain.Comment) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
func (m *mockCommentRepo) FindByID(ctx context.Context, id string) (*domain.Comment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Comment), args.Error(1)
}
func (m *mockCommentRepo) ListByCase(ctx context.Context, caseID string) ([]*domain.Comment, error) {
	args := m.Called(ctx, caseID)
	return args.Get(0).([]*domain.Comment), args.Error(1)
}
func (m *mockCommentRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func newTestCaseService(caseRepo *mockCaseRepo, wfRepo *mockWorkflowRepo, commentRepo *mockCommentRepo) ports.CaseService {
	logger := zap.NewNop()
	return application.NewCaseService(caseRepo, wfRepo, commentRepo, logger)
}

func TestCreateCase_Success(t *testing.T) {
	caseRepo := new(mockCaseRepo)
	wfRepo := new(mockWorkflowRepo)
	commentRepo := new(mockCommentRepo)

	caseRepo.On("NextCaseNumber", mock.Anything, "tenant-1").Return("CASE-TENAN-00001", nil)
	caseRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Case")).Return(nil)

	svc := newTestCaseService(caseRepo, wfRepo, commentRepo)
	req := ports.CreateCaseRequest{
		Title:       "Test Case",
		Description: "A test case",
		Priority:    domain.PriorityHigh,
	}
	c, err := svc.CreateCase(context.Background(), "tenant-1", "reporter-1", req)
	require.NoError(t, err)
	assert.Equal(t, "CASE-TENAN-00001", c.CaseNumber)
	assert.Equal(t, domain.CaseStatusNew, c.Status)
	assert.Equal(t, domain.PriorityHigh, c.Priority)
	assert.Equal(t, "tenant-1", c.TenantID)
	assert.Equal(t, "reporter-1", c.ReporterID)
	assert.NotEmpty(t, c.ID)
	caseRepo.AssertExpectations(t)
}

func TestTransitionCase_ValidTransition(t *testing.T) {
	caseRepo := new(mockCaseRepo)
	wfRepo := new(mockWorkflowRepo)
	commentRepo := new(mockCommentRepo)

	wfID := "wf-1"
	existingCase := &domain.Case{
		ID:         "case-1",
		TenantID:   "tenant-1",
		Status:     domain.CaseStatusNew,
		WorkflowID: &wfID,
		CreatedAt:  time.Now(),
	}
	wf := &domain.WorkflowTemplate{
		ID: wfID,
		Transitions: []domain.Transition{
			{From: domain.CaseStatusNew, To: domain.CaseStatusOpen, Label: "Open"},
		},
	}

	caseRepo.On("FindByID", mock.Anything, "case-1").Return(existingCase, nil)
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wf, nil)
	caseRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Case")).Return(nil)
	caseRepo.On("RecordTransition", mock.Anything, mock.AnythingOfType("*domain.CaseTransition")).Return(nil)

	svc := newTestCaseService(caseRepo, wfRepo, commentRepo)
	result, err := svc.TransitionCase(context.Background(), "case-1", ports.TransitionRequest{
		ToStatus:    domain.CaseStatusOpen,
		Reason:      "Starting work",
		PerformedBy: "user-1",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.CaseStatusOpen, result.Status)
	caseRepo.AssertExpectations(t)
}

func TestTransitionCase_InvalidTransition(t *testing.T) {
	caseRepo := new(mockCaseRepo)
	wfRepo := new(mockWorkflowRepo)
	commentRepo := new(mockCommentRepo)

	wfID := "wf-1"
	existingCase := &domain.Case{
		ID:         "case-1",
		Status:     domain.CaseStatusNew,
		WorkflowID: &wfID,
	}
	wf := &domain.WorkflowTemplate{
		ID: wfID,
		Transitions: []domain.Transition{
			{From: domain.CaseStatusNew, To: domain.CaseStatusOpen, Label: "Open"},
		},
	}

	caseRepo.On("FindByID", mock.Anything, "case-1").Return(existingCase, nil)
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wf, nil)

	svc := newTestCaseService(caseRepo, wfRepo, commentRepo)
	_, err := svc.TransitionCase(context.Background(), "case-1", ports.TransitionRequest{
		ToStatus:    domain.CaseStatusClosed, // Not allowed from new
		PerformedBy: "user-1",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestAssignCase_Success(t *testing.T) {
	caseRepo := new(mockCaseRepo)
	wfRepo := new(mockWorkflowRepo)
	commentRepo := new(mockCommentRepo)

	existingCase := &domain.Case{
		ID:       "case-1",
		TenantID: "tenant-1",
		Status:   domain.CaseStatusOpen,
	}
	caseRepo.On("FindByID", mock.Anything, "case-1").Return(existingCase, nil)
	caseRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Case")).Return(nil)

	svc := newTestCaseService(caseRepo, wfRepo, commentRepo)
	result, err := svc.AssignCase(context.Background(), "case-1", "assignee-99")
	require.NoError(t, err)
	require.NotNil(t, result.AssigneeID)
	assert.Equal(t, "assignee-99", *result.AssigneeID)
	caseRepo.AssertExpectations(t)
}

func TestAddComment_Success(t *testing.T) {
	caseRepo := new(mockCaseRepo)
	wfRepo := new(mockWorkflowRepo)
	commentRepo := new(mockCommentRepo)

	existingCase := &domain.Case{
		ID:       "case-1",
		TenantID: "tenant-1",
	}
	caseRepo.On("FindByID", mock.Anything, "case-1").Return(existingCase, nil)
	commentRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Comment")).Return(nil)

	svc := newTestCaseService(caseRepo, wfRepo, commentRepo)
	comment, err := svc.AddComment(context.Background(), "case-1", "author-1", "This is a test comment")
	require.NoError(t, err)
	assert.Equal(t, "case-1", comment.CaseID)
	assert.Equal(t, "author-1", comment.AuthorID)
	assert.Equal(t, "This is a test comment", comment.Body)
	assert.NotEmpty(t, comment.ID)
	commentRepo.AssertExpectations(t)
}
