package domain

import "time"

// UserProfile stores profile information for a user within a specific tenant.
type UserProfile struct {
	UserID         string
	TenantID       string
	OrganizationID *string
	DisplayName    string
	AvatarURL      string
	Timezone       string
	Locale         string
	Metadata       map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
