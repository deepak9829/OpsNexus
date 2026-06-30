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

type taskService struct {
	taskRepo ports.TaskRepository
	caseRepo ports.CaseRepository
	logger   *zap.Logger
}

func NewTaskService(taskRepo ports.TaskRepository, caseRepo ports.CaseRepository, logger *zap.Logger) ports.TaskService {
	return &taskService{
		taskRepo: taskRepo,
		caseRepo: caseRepo,
		logger:   logger,
	}
}

func (s *taskService) CreateTask(ctx context.Context, caseID string, req ports.CreateTaskRequest) (*domain.Task, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}

	c, err := s.caseRepo.FindByID(ctx, caseID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	task := &domain.Task{
		ID:          uuid.NewString(),
		CaseID:      caseID,
		TenantID:    c.TenantID,
		Title:       req.Title,
		Description: req.Description,
		Status:      domain.TaskStatusTodo,
		AssigneeID:  req.AssigneeID,
		DueAt:       req.DueAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}
	s.logger.Info("task created", zap.String("taskID", task.ID), zap.String("caseID", caseID))
	return task, nil
}

func (s *taskService) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	return s.taskRepo.FindByID(ctx, id)
}

func (s *taskService) UpdateTask(ctx context.Context, id string, req ports.UpdateTaskRequest) (*domain.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, fmt.Errorf("%w: title cannot be empty", domain.ErrInvalidInput)
		}
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = *req.Status
		if *req.Status == domain.TaskStatusDone && task.CompletedAt == nil {
			now := time.Now().UTC()
			task.CompletedAt = &now
		}
	}
	if req.AssigneeID != nil {
		task.AssigneeID = req.AssigneeID
	}
	if req.DueAt != nil {
		task.DueAt = req.DueAt
	}

	task.UpdatedAt = time.Now().UTC()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("updating task: %w", err)
	}
	return task, nil
}

func (s *taskService) CompleteTask(ctx context.Context, id string) (*domain.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	task.Status = domain.TaskStatusDone
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("completing task: %w", err)
	}
	return task, nil
}

func (s *taskService) ListByCase(ctx context.Context, caseID string) ([]*domain.Task, error) {
	return s.taskRepo.ListByCase(ctx, caseID)
}
