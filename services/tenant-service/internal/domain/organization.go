package domain

import "time"

// Organization is a structural unit within a tenant (department, team, division, …).
type Organization struct {
	ID        string
	TenantID  string
	Name      string
	Type      string // department | team | division
	ParentID  *string
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}
