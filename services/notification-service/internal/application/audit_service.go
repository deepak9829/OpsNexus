package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opsnexus/notification-service/internal/domain"
	"github.com/opsnexus/notification-service/internal/ports"
	"go.uber.org/zap"
)

type auditService struct {
	repo   ports.AuditRepository
	logger *zap.Logger
}

func NewAuditService(repo ports.AuditRepository, logger *zap.Logger) ports.AuditService {
	return &auditService{repo: repo, logger: logger}
}

func (s *auditService) Record(ctx context.Context, req ports.RecordAuditRequest) (*domain.AuditEvent, error) {
	event := &domain.AuditEvent{
		ID:         uuid.New().String(),
		TenantID:   req.TenantID,
		ActorID:    req.ActorID,
		ActorEmail: req.ActorEmail,
		Action:     req.Action,
		Resource:   req.Resource,
		ResourceID: req.ResourceID,
		OldValue:   req.OldValue,
		NewValue:   req.NewValue,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		Timestamp:  time.Now().UTC(),
	}

	if err := s.repo.Record(ctx, event); err != nil {
		s.logger.Error("failed to record audit event", zap.Error(err))
		return nil, err
	}

	s.logger.Info("audit event recorded",
		zap.String("id", event.ID),
		zap.String("action", event.Action),
		zap.String("actor_id", event.ActorID),
		zap.String("resource", event.Resource),
		zap.String("resource_id", event.ResourceID),
	)
	return event, nil
}

func (s *auditService) GetEvent(ctx context.Context, tenantID, id string) (*domain.AuditEvent, error) {
	return s.repo.FindByID(ctx, tenantID, id)
}

func (s *auditService) QueryEvents(ctx context.Context, tenantID string, filter ports.AuditFilter, page, limit int) ([]*domain.AuditEvent, int64, error) {
	return s.repo.Query(ctx, tenantID, filter, page, limit)
}
