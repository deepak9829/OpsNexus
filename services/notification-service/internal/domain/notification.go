package domain

import "time"

type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeSuccess NotificationType = "success"
)

type NotificationChannel string

const (
	ChannelInApp NotificationChannel = "in_app"
	ChannelEmail NotificationChannel = "email"
	ChannelSMS   NotificationChannel = "sms"
)

type Notification struct {
	ID        string              `json:"id"`
	TenantID  string              `json:"tenant_id"`
	UserID    string              `json:"user_id"`
	Type      NotificationType    `json:"type"`
	Title     string              `json:"title"`
	Body      string              `json:"body"`
	Channel   NotificationChannel `json:"channel"`
	Read      bool                `json:"read"`
	ReadAt    *time.Time          `json:"read_at,omitempty"`
	Metadata  map[string]string   `json:"metadata,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
}
