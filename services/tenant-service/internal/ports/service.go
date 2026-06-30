package ports

import (
	"context"

	"github.com/opsnexus/tenant-service/internal/domain"
)

// ─── Request / Response types ────────────────────────────────────────────────

// CreateTenantRequest carries the data needed to provision a new tenant.
type CreateTenantRequest struct {
	Name string
	Slug string
	Plan domain.TenantPlan
}

// UpdateTenantRequest carries fields that may be patched on an existing tenant.
type UpdateTenantRequest struct {
	Name   *string
	Plan   *domain.TenantPlan
	Status *domain.TenantStatus
}

// CreateOrgRequest carries the data needed to create a new organization.
type CreateOrgRequest struct {
	Name     string
	Type     string
	ParentID *string
}

// UpdateOrgRequest carries fields that may be patched on an existing organization.
type UpdateOrgRequest struct {
	Name     *string
	Type     *string
	ParentID *string
}

// CreateProfileRequest carries the data needed to create a user profile.
type CreateProfileRequest struct {
	UserID         string
	TenantID       string
	OrganizationID *string
	DisplayName    string
	Timezone       string
	Locale         string
}

// UpdateProfileRequest carries fields that may be patched on an existing profile.
type UpdateProfileRequest struct {
	DisplayName    *string
	AvatarURL      *string
	Timezone       *string
	Locale         *string
	OrganizationID *string
}

// ─── Service interfaces ───────────────────────────────────────────────────────

// TenantService defines business operations on Tenant aggregates.
type TenantService interface {
	CreateTenant(ctx context.Context, req CreateTenantRequest) (*domain.Tenant, error)
	GetTenant(ctx context.Context, id string) (*domain.Tenant, error)
	UpdateTenant(ctx context.Context, id string, req UpdateTenantRequest) (*domain.Tenant, error)
	DeactivateTenant(ctx context.Context, id string) error
	ListTenants(ctx context.Context, page, limit int) ([]*domain.Tenant, int64, error)
	GetSettings(ctx context.Context, tenantID string) (*domain.TenantSettings, error)
	UpdateSettings(ctx context.Context, tenantID string, settings domain.TenantSettings) error
}

// OrganizationService defines business operations on Organization entities.
type OrganizationService interface {
	CreateOrganization(ctx context.Context, tenantID string, req CreateOrgRequest) (*domain.Organization, error)
	GetOrganization(ctx context.Context, id string) (*domain.Organization, error)
	ListOrganizations(ctx context.Context, tenantID string) ([]*domain.Organization, error)
	UpdateOrganization(ctx context.Context, id string, req UpdateOrgRequest) (*domain.Organization, error)
}

// UserProfileService defines business operations on UserProfile entities.
type UserProfileService interface {
	CreateProfile(ctx context.Context, req CreateProfileRequest) (*domain.UserProfile, error)
	GetProfile(ctx context.Context, userID string) (*domain.UserProfile, error)
	UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*domain.UserProfile, error)
	ListProfiles(ctx context.Context, tenantID string, page, limit int) ([]*domain.UserProfile, int64, error)
}
