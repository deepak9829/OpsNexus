package ports

import (
	"context"

	"github.com/opsnexus/tenant-service/internal/domain"
)

// TenantRepository defines persistence operations for Tenant aggregates.
type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	FindByID(ctx context.Context, id string) (*domain.Tenant, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Deactivate(ctx context.Context, id string) error
	List(ctx context.Context, page, limit int) ([]*domain.Tenant, int64, error)
}

// OrganizationRepository defines persistence operations for Organization entities.
type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	FindByID(ctx context.Context, id string) (*domain.Organization, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.Organization, error)
	Update(ctx context.Context, org *domain.Organization) error
	Delete(ctx context.Context, id string) error
}

// UserProfileRepository defines persistence operations for UserProfile entities.
type UserProfileRepository interface {
	Create(ctx context.Context, profile *domain.UserProfile) error
	FindByUserID(ctx context.Context, userID string) (*domain.UserProfile, error)
	Update(ctx context.Context, profile *domain.UserProfile) error
	ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.UserProfile, int64, error)
}
