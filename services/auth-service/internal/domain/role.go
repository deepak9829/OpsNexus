package domain

import "time"

type Role struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	Permissions []Permission
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Permission struct {
	ID       string
	Resource string
	Action   string // read, write, delete, admin
}

var SystemRoles = []string{"super_admin", "admin", "operator", "viewer"}
