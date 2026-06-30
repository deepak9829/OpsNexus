package ports

import (
	"context"

	"github.com/opsnexus/auth-service/internal/domain"
)

type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req LoginRequest) (*domain.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	ValidateToken(ctx context.Context, accessToken string) (*Claims, error)
	GetCurrentUser(ctx context.Context, userID string) (*domain.User, error)
	AssignRole(ctx context.Context, userID, roleID string) error
	ListUsers(ctx context.Context, tenantID string, page, limit int) ([]*domain.User, int64, error)
	UpdateUserStatus(ctx context.Context, userID, status string) error
}

type RegisterRequest struct {
	TenantID  string
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type LoginRequest struct {
	Email    string
	Password string
}

type Claims struct {
	UserID   string
	TenantID string
	Email    string
	Roles    []string
}
