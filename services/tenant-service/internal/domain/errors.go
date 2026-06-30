package domain

import "errors"

// Sentinel errors for the tenant domain.
var (
	ErrTenantNotFound       = errors.New("tenant not found")
	ErrTenantAlreadyExists  = errors.New("tenant already exists")
	ErrSlugTaken            = errors.New("slug is already taken")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrProfileNotFound      = errors.New("user profile not found")
	ErrInvalidInput         = errors.New("invalid input")
)
