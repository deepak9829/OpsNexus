package ports

import (
	"context"
	"time"

	"github.com/opsnexus/notification-service/internal/domain"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) error
	FindByID(ctx context.Context, tenantID, id string) (*domain.Notification, error)
	ListByUser(ctx context.Context, tenantID, userID string, page, limit int) ([]*domain.Notification, int64, error)
	MarkRead(ctx context.Context, tenantID, id string) error
	MarkAllRead(ctx context.Context, tenantID, userID string) error
	Delete(ctx context.Context, tenantID, id string) error
}

type AuditFilter struct {
	ActorID    *string
	Action     *string
	Resource   *string
	ResourceID *string
	From       *time.Time
	To         *time.Time
}

type AuditRepository interface {
	Record(ctx context.Context, event *domain.AuditEvent) error
	FindByID(ctx context.Context, tenantID, id string) (*domain.AuditEvent, error)
	Query(ctx context.Context, tenantID string, filter AuditFilter, page, limit int) ([]*domain.AuditEvent, int64, error)
}
