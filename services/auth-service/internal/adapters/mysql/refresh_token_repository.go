package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/opsnexus/auth-service/internal/domain"
	"github.com/opsnexus/auth-service/internal/ports"
	"gorm.io/gorm"
)

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) ports.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	model := refreshTokenModel{
		ID:        token.ID,
		UserID:    token.UserID,
		TenantID:  token.TenantID,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
		Revoked:   token.Revoked,
		CreatedAt: token.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("creating refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	var model refreshTokenModel
	err := r.db.WithContext(ctx).First(&model, "token = ?", token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTokenInvalid
		}
		return nil, fmt.Errorf("finding refresh token: %w", err)
	}
	return &domain.RefreshToken{
		ID:        model.ID,
		UserID:    model.UserID,
		TenantID:  model.TenantID,
		Token:     model.Token,
		ExpiresAt: model.ExpiresAt,
		Revoked:   model.Revoked,
		CreatedAt: model.CreatedAt,
	}, nil
}

func (r *refreshTokenRepository) RevokeByUserID(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).
		Model(&refreshTokenModel{}).
		Where("user_id = ? AND revoked = false", userID).
		Update("revoked", true).Error; err != nil {
		return fmt.Errorf("revoking tokens by user id: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeToken(ctx context.Context, token string) error {
	if err := r.db.WithContext(ctx).
		Model(&refreshTokenModel{}).
		Where("token = ?", token).
		Update("revoked", true).Error; err != nil {
		return fmt.Errorf("revoking token: %w", err)
	}
	return nil
}
