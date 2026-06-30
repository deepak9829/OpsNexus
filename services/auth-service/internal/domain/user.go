package domain

import (
	"time"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

type User struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Status       UserStatus
	Roles        []Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

func (u *User) HasRole(name string) bool {
	for _, r := range u.Roles {
		if r.Name == name {
			return true
		}
	}
	return false
}

func (u *User) HasPermission(resource, action string) bool {
	for _, r := range u.Roles {
		for _, p := range r.Permissions {
			if p.Resource == resource && (p.Action == action || p.Action == "admin") {
				return true
			}
		}
	}
	return false
}
