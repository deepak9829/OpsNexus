package domain

import "time"

// TenantStatus represents the lifecycle state of a tenant.
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended"
)

// TenantPlan represents the billing/feature plan of a tenant.
type TenantPlan string

const (
	PlanFree       TenantPlan = "free"
	PlanPro        TenantPlan = "pro"
	PlanEnterprise TenantPlan = "enterprise"
)

// Tenant is the root aggregate for a tenant account.
type Tenant struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Slug           string       `json:"slug"`
	Plan           TenantPlan   `json:"plan"`
	Status         TenantStatus `json:"status"`
	Settings       TenantSettings `json:"settings"`
	MaxUsers       int          `json:"max_users"`
	AllowedDomains []string     `json:"allowed_domains"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// TenantSettings holds configurable behaviour for a tenant.
type TenantSettings struct {
	MaxUsers          int                     `json:"max_users"`
	AllowedDomains    []string                `json:"allowed_domains"`
	Features          map[string]bool         `json:"features"`
	NotificationPrefs NotificationPreferences `json:"notification_prefs"`
}

// NotificationPreferences controls which notification channels are active.
type NotificationPreferences struct {
	EmailEnabled bool `json:"email_enabled"`
	SMSEnabled   bool `json:"sms_enabled"`
	InAppEnabled bool `json:"in_app_enabled"`
}
