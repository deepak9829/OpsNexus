package ports

import (
	"context"

	"github.com/opsnexus/auth-service/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByTenantAndEmail(ctx context.Context, tenantID, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.User, int64, error)
}

type RoleRepository interface {
	Create(ctx context.Context, role *domain.Role) error
	FindByID(ctx context.Context, id string) (*domain.Role, error)
	FindByName(ctx context.Context, tenantID, name string) (*domain.Role, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.Role, error)
	AssignToUser(ctx context.Context, userID, roleID string) error
	RevokeFromUser(ctx context.Context, userID, roleID string) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error)
	RevokeByUserID(ctx context.Context, userID string) error
	RevokeToken(ctx context.Context, token string) error
}
