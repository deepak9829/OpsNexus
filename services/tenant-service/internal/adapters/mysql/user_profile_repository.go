package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/opsnexus/tenant-service/internal/domain"
	"github.com/opsnexus/tenant-service/internal/ports"
)

// userProfileModel is the GORM representation of the user_profiles table.
type userProfileModel struct {
	UserID         string    `gorm:"primaryKey;type:varchar(36)"`
	TenantID       string    `gorm:"type:varchar(36);not null;index:idx_profile_tenant_id"`
	OrganizationID *string   `gorm:"type:varchar(36)"`
	DisplayName    string    `gorm:"type:varchar(255)"`
	AvatarURL      string    `gorm:"type:varchar(512)"`
	Timezone       string    `gorm:"type:varchar(100);default:UTC"`
	Locale         string    `gorm:"type:varchar(20);default:en"`
	Metadata       string    `gorm:"type:text"` // JSON-encoded map[string]any
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (userProfileModel) TableName() string { return "user_profiles" }

// userProfileRepository implements ports.UserProfileRepository.
type userProfileRepository struct {
	db *gorm.DB
}

// NewUserProfileRepository constructs a GORM-backed UserProfileRepository.
func NewUserProfileRepository(db *gorm.DB) ports.UserProfileRepository {
	return &userProfileRepository{db: db}
}

func (r *userProfileRepository) Create(ctx context.Context, p *domain.UserProfile) error {
	m, err := domainToProfileModel(p)
	if err != nil {
		return err
	}
	if res := r.db.WithContext(ctx).Create(m); res.Error != nil {
		return fmt.Errorf("insert user_profile: %w", res.Error)
	}
	return nil
}

func (r *userProfileRepository) FindByUserID(ctx context.Context, userID string) (*domain.UserProfile, error) {
	var m userProfileModel
	res := r.db.WithContext(ctx).First(&m, "user_id = ?", userID)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, fmt.Errorf("find profile by user_id: %w", res.Error)
	}
	return profileModelToDomain(&m)
}

func (r *userProfileRepository) Update(ctx context.Context, p *domain.UserProfile) error {
	m, err := domainToProfileModel(p)
	if err != nil {
		return err
	}
	if res := r.db.WithContext(ctx).Save(m); res.Error != nil {
		return fmt.Errorf("update user_profile: %w", res.Error)
	}
	return nil
}

func (r *userProfileRepository) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.UserProfile, int64, error) {
	var models []userProfileModel
	var total int64
	offset := (page - 1) * limit

	if err := r.db.WithContext(ctx).Model(&userProfileModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count profiles: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list profiles by tenant: %w", err)
	}

	profiles := make([]*domain.UserProfile, 0, len(models))
	for i := range models {
		p, err := profileModelToDomain(&models[i])
		if err != nil {
			return nil, 0, err
		}
		profiles = append(profiles, p)
	}

	return profiles, total, nil
}

// ─── mapping helpers ──────────────────────────────────────────────────────────

func domainToProfileModel(p *domain.UserProfile) (*userProfileModel, error) {
	meta := p.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	return &userProfileModel{
		UserID:         p.UserID,
		TenantID:       p.TenantID,
		OrganizationID: p.OrganizationID,
		DisplayName:    p.DisplayName,
		AvatarURL:      p.AvatarURL,
		Timezone:       p.Timezone,
		Locale:         p.Locale,
		Metadata:       string(metaJSON),
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}, nil
}

func profileModelToDomain(m *userProfileModel) (*domain.UserProfile, error) {
	var metadata map[string]any
	if m.Metadata != "" {
		if err := json.Unmarshal([]byte(m.Metadata), &metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}

	return &domain.UserProfile{
		UserID:         m.UserID,
		TenantID:       m.TenantID,
		OrganizationID: m.OrganizationID,
		DisplayName:    m.DisplayName,
		AvatarURL:      m.AvatarURL,
		Timezone:       m.Timezone,
		Locale:         m.Locale,
		Metadata:       metadata,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}, nil
}
