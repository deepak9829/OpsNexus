package ports

import (
	"context"

	"github.com/opsnexus/notification-service/internal/domain"
)

type SendNotificationRequest struct {
	TenantID string
	UserID   string
	Type     domain.NotificationType
	Title    string
	Body     string
	Channel  domain.NotificationChannel
	Metadata map[string]string
}

type RecordAuditRequest struct {
	TenantID   string
	ActorID    string
	ActorEmail string
	Action     string
	Resource   string
	ResourceID string
	OldValue   map[string]any
	NewValue   map[string]any
	IPAddress  string
	UserAgent  string
}

type NotificationService interface {
	Send(ctx context.Context, req SendNotificationRequest) (*domain.Notification, error)
	SendBulk(ctx context.Context, reqs []SendNotificationRequest) error
	GetNotification(ctx context.Context, tenantID, id string) (*domain.Notification, error)
	ListForUser(ctx context.Context, tenantID, userID string, page, limit int) ([]*domain.Notification, int64, error)
	MarkRead(ctx context.Context, tenantID, id string) error
	MarkAllRead(ctx context.Context, tenantID, userID string) error
	Dismiss(ctx context.Context, tenantID, id string) error
}

type AuditService interface {
	Record(ctx context.Context, req RecordAuditRequest) (*domain.AuditEvent, error)
	GetEvent(ctx context.Context, tenantID, id string) (*domain.AuditEvent, error)
	QueryEvents(ctx context.Context, tenantID string, filter AuditFilter, page, limit int) ([]*domain.AuditEvent, int64, error)
}
