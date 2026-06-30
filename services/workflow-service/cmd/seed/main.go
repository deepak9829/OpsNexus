package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/opsnexus/workflow-service/internal/adapters/mysql"
	"github.com/opsnexus/workflow-service/internal/application"
	"github.com/opsnexus/workflow-service/internal/config"
	"github.com/opsnexus/workflow-service/internal/domain"
	"github.com/opsnexus/workflow-service/internal/ports"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	db, err := mysql.NewDB(mysql.Config{
		DSN:             cfg.Database.DSN,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		logger.Fatal("connecting to database", zap.Error(err))
	}

	// Auto-migrate first
	if err := mysql.AutoMigrate(db); err != nil {
		logger.Fatal("migration failed", zap.Error(err))
	}

	// Repositories
	caseRepo := mysql.NewCaseRepository(db)
	taskRepo := mysql.NewTaskRepository(db)
	wfRepo := mysql.NewWorkflowTemplateRepository(db)
	commentRepo := mysql.NewCommentRepository(db)

	// Services
	caseService := application.NewCaseService(caseRepo, wfRepo, commentRepo, logger)
	taskService := application.NewTaskService(taskRepo, caseRepo, logger)
	workflowService := application.NewWorkflowService(wfRepo, logger)

	ctx := context.Background()
	tenantID := "tenant-" + uuid.NewString()[:8]

	// 1. Create default IT Support workflow template
	wf, err := workflowService.CreateTemplate(ctx, tenantID, ports.CreateWorkflowTemplateRequest{
		Name:        "IT Support Workflow",
		Description: "Standard IT support case workflow with SLA tracking",
		States: []string{
			string(domain.CaseStatusNew),
			string(domain.CaseStatusOpen),
			string(domain.CaseStatusInProgress),
			string(domain.CaseStatusPending),
			string(domain.CaseStatusResolved),
			string(domain.CaseStatusClosed),
		},
		Transitions: []domain.Transition{
			{From: domain.CaseStatusNew, To: domain.CaseStatusOpen, Label: "Open"},
			{From: domain.CaseStatusOpen, To: domain.CaseStatusInProgress, Label: "Start Work"},
			{From: domain.CaseStatusOpen, To: domain.CaseStatusClosed, Label: "Close (Duplicate/Invalid)"},
			{From: domain.CaseStatusInProgress, To: domain.CaseStatusPending, Label: "Waiting on Customer"},
			{From: domain.CaseStatusInProgress, To: domain.CaseStatusResolved, Label: "Resolve"},
			{From: domain.CaseStatusPending, To: domain.CaseStatusInProgress, Label: "Resume"},
			{From: domain.CaseStatusPending, To: domain.CaseStatusResolved, Label: "Resolve"},
			{From: domain.CaseStatusResolved, To: domain.CaseStatusClosed, Label: "Close"},
			{From: domain.CaseStatusResolved, To: domain.CaseStatusOpen, Label: "Reopen"},
			{From: domain.CaseStatusClosed, To: domain.CaseStatusOpen, Label: "Reopen"},
		},
		DefaultPriority: domain.PriorityMedium,
		SLAHours:        24,
	})
	if err != nil {
		logger.Fatal("creating workflow template", zap.Error(err))
	}
	logger.Info("created workflow template", zap.String("id", wf.ID), zap.String("name", wf.Name))

	reporterID := "user-" + uuid.NewString()[:8]
	assigneeID := "user-" + uuid.NewString()[:8]

	// 2. Create 3 sample cases with different statuses
	type seedCase struct {
		title       string
		description string
		priority    domain.CasePriority
		status      domain.CaseStatus
		tags        []string
	}

	seedCases := []seedCase{
		{
			title:       "VPN Access Issue - Unable to connect from home",
			description: "User reports VPN client fails to connect after recent password change. Error: authentication failed.",
			priority:    domain.PriorityHigh,
			status:      domain.CaseStatusNew,
			tags:        []string{"vpn", "access", "remote-work"},
		},
		{
			title:       "Laptop performance degradation after Windows update",
			description: "Multiple users in Finance department reporting slow performance after KB5034122 update applied on 2024-01-15.",
			priority:    domain.PriorityCritical,
			status:      domain.CaseStatusInProgress,
			tags:        []string{"laptop", "windows", "performance", "finance"},
		},
		{
			title:       "Email signature not displaying correctly in Outlook",
			description: "HTML email signature is rendering as plain text in Outlook 365 on Windows 11.",
			priority:    domain.PriorityLow,
			status:      domain.CaseStatusResolved,
			tags:        []string{"email", "outlook", "signature"},
		},
	}

	for _, sc := range seedCases {
		wfID := wf.ID
		c, err := caseService.CreateCase(ctx, tenantID, reporterID, ports.CreateCaseRequest{
			Title:       sc.title,
			Description: sc.description,
			Priority:    sc.priority,
			WorkflowID:  &wfID,
			Tags:        sc.tags,
		})
		if err != nil {
			logger.Fatal("creating case", zap.Error(err))
		}
		logger.Info("created case", zap.String("id", c.ID), zap.String("number", c.CaseNumber))

		// Transition to target status
		transitions := transitionPath(domain.CaseStatusNew, sc.status)
		for _, toStatus := range transitions {
			_, err := caseService.TransitionCase(ctx, c.ID, ports.TransitionRequest{
				ToStatus:    toStatus,
				Reason:      "Seed data transition",
				PerformedBy: assigneeID,
			})
			if err != nil {
				logger.Warn("transition failed", zap.String("caseID", c.ID), zap.String("to", string(toStatus)), zap.Error(err))
			}
		}

		// Assign the case
		_, err = caseService.AssignCase(ctx, c.ID, assigneeID)
		if err != nil {
			logger.Warn("assigning case", zap.Error(err))
		}

		// Create tasks for each case
		dueAt := time.Now().Add(48 * time.Hour)
		task1, err := taskService.CreateTask(ctx, c.ID, ports.CreateTaskRequest{
			Title:       "Initial investigation and diagnosis",
			Description: "Gather logs, reproduce the issue, identify root cause",
			AssigneeID:  &assigneeID,
			DueAt:       &dueAt,
		})
		if err != nil {
			logger.Warn("creating task1", zap.Error(err))
		} else {
			logger.Info("created task", zap.String("taskID", task1.ID))
		}

		_, err = taskService.CreateTask(ctx, c.ID, ports.CreateTaskRequest{
			Title:       "Apply fix and verify resolution",
			Description: "Implement fix, test with affected user, document solution",
			AssigneeID:  &assigneeID,
		})
		if err != nil {
			logger.Warn("creating task2", zap.Error(err))
		}

		// Add a comment
		_, err = caseService.AddComment(ctx, c.ID, reporterID, "Case has been logged and is under review by the IT support team.")
		if err != nil {
			logger.Warn("adding comment", zap.Error(err))
		}
	}

	logger.Info("seed completed", zap.String("tenantID", tenantID))
}

// transitionPath returns the ordered list of statuses to transition through from new to target
func transitionPath(from, to domain.CaseStatus) []domain.CaseStatus {
	if from == to {
		return nil
	}
	// Ordered path through the standard workflow
	orderedPath := []domain.CaseStatus{
		domain.CaseStatusOpen,
		domain.CaseStatusInProgress,
		domain.CaseStatusResolved,
		domain.CaseStatusClosed,
	}

	var result []domain.CaseStatus
	collecting := (from == domain.CaseStatusNew)
	for _, s := range orderedPath {
		if s == from {
			collecting = true
			continue
		}
		if collecting {
			result = append(result, s)
		}
		if s == to {
			break
		}
	}
	return result
}
