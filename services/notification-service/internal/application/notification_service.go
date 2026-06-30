package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opsnexus/notification-service/internal/domain"
	"github.com/opsnexus/notification-service/internal/ports"
	"go.uber.org/zap"
)

type notificationService struct {
	repo   ports.NotificationRepository
	logger *zap.Logger
}

func NewNotificationService(repo ports.NotificationRepository, logger *zap.Logger) ports.NotificationService {
	return &notificationService{repo: repo, logger: logger}
}

func (s *notificationService) Send(ctx context.Context, req ports.SendNotificationRequest) (*domain.Notification, error) {
	n := &domain.Notification{
		ID:        uuid.New().String(),
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		Type:      req.Type,
		Title:     req.Title,
		Body:      req.Body,
		Channel:   req.Channel,
		Read:      false,
		Metadata:  req.Metadata,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, n); err != nil {
		s.logger.Error("failed to create notification", zap.Error(err))
		return nil, err
	}

	s.logger.Info("notification sent",
		zap.String("id", n.ID),
		zap.String("tenant_id", n.TenantID),
		zap.String("user_id", n.UserID),
		zap.String("type", string(n.Type)),
	)
	return n, nil
}

func (s *notificationService) SendBulk(ctx context.Context, reqs []ports.SendNotificationRequest) error {
	for _, req := range reqs {
		if _, err := s.Send(ctx, req); err != nil {
			s.logger.Error("failed to send bulk notification", zap.Error(err))
			return err
		}
	}
	return nil
}

func (s *notificationService) GetNotification(ctx context.Context, tenantID, id string) (*domain.Notification, error) {
	return s.repo.FindByID(ctx, tenantID, id)
}

func (s *notificationService) ListForUser(ctx context.Context, tenantID, userID string, page, limit int) ([]*domain.Notification, int64, error) {
	return s.repo.ListByUser(ctx, tenantID, userID, page, limit)
}

func (s *notificationService) MarkRead(ctx context.Context, tenantID, id string) error {
	if err := s.repo.MarkRead(ctx, tenantID, id); err != nil {
		return err
	}
	s.logger.Info("notification marked read", zap.String("id", id))
	return nil
}

func (s *notificationService) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	return s.repo.MarkAllRead(ctx, tenantID, userID)
}

func (s *notificationService) Dismiss(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}
