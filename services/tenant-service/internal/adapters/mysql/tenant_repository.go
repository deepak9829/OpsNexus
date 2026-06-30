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

// tenantModel is the GORM representation of the tenants table.
type tenantModel struct {
	ID             string    `gorm:"primaryKey;type:varchar(36)"`
	Name           string    `gorm:"type:varchar(255);not null"`
	Slug           string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Plan           string    `gorm:"type:varchar(50);not null;default:free"`
	Status         string    `gorm:"type:varchar(20);not null;default:active"`
	MaxUsers       int       `gorm:"not null;default:10"`
	AllowedDomains string    `gorm:"type:text"` // JSON-encoded []string
	Features       string    `gorm:"type:text"` // JSON-encoded map[string]bool
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (tenantModel) TableName() string { return "tenants" }

// tenantRepository implements ports.TenantRepository using GORM.
type tenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository constructs a GORM-backed TenantRepository.
func NewTenantRepository(db *gorm.DB) ports.TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) Create(ctx context.Context, t *domain.Tenant) error {
	m, err := domainToTenantModel(t)
	if err != nil {
		return err
	}
	if res := r.db.WithContext(ctx).Create(m); res.Error != nil {
		return fmt.Errorf("insert tenant: %w", res.Error)
	}
	return nil
}

func (r *tenantRepository) FindByID(ctx context.Context, id string) (*domain.Tenant, error) {
	var m tenantModel
	res := r.db.WithContext(ctx).First(&m, "id = ?", id)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("find tenant by id: %w", res.Error)
	}
	return tenantModelToDomain(&m)
}

func (r *tenantRepository) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	var m tenantModel
	res := r.db.WithContext(ctx).First(&m, "slug = ?", slug)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("find tenant by slug: %w", res.Error)
	}
	return tenantModelToDomain(&m)
}

func (r *tenantRepository) Update(ctx context.Context, t *domain.Tenant) error {
	m, err := domainToTenantModel(t)
	if err != nil {
		return err
	}
	res := r.db.WithContext(ctx).Save(m)
	if res.Error != nil {
		return fmt.Errorf("update tenant: %w", res.Error)
	}
	return nil
}

func (r *tenantRepository) Deactivate(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Model(&tenantModel{}).
		Where("id = ?", id).
		Update("status", string(domain.TenantStatusInactive))
	if res.Error != nil {
		return fmt.Errorf("deactivate tenant: %w", res.Error)
	}
	return nil
}

func (r *tenantRepository) List(ctx context.Context, page, limit int) ([]*domain.Tenant, int64, error) {
	var models []tenantModel
	var total int64

	offset := (page - 1) * limit

	if err := r.db.WithContext(ctx).Model(&tenantModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}

	tenants := make([]*domain.Tenant, 0, len(models))
	for i := range models {
		t, err := tenantModelToDomain(&models[i])
		if err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, t)
	}

	return tenants, total, nil
}

// ─── mapping helpers ──────────────────────────────────────────────────────────

func domainToTenantModel(t *domain.Tenant) (*tenantModel, error) {
	domainsJSON, err := json.Marshal(t.AllowedDomains)
	if err != nil {
		return nil, fmt.Errorf("marshal allowed_domains: %w", err)
	}
	featuresJSON, err := json.Marshal(t.Settings.Features)
	if err != nil {
		return nil, fmt.Errorf("marshal features: %w", err)
	}

	return &tenantModel{
		ID:             t.ID,
		Name:           t.Name,
		Slug:           t.Slug,
		Plan:           string(t.Plan),
		Status:         string(t.Status),
		MaxUsers:       t.MaxUsers,
		AllowedDomains: string(domainsJSON),
		Features:       string(featuresJSON),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}, nil
}

func tenantModelToDomain(m *tenantModel) (*domain.Tenant, error) {
	var allowedDomains []string
	if m.AllowedDomains != "" {
		if err := json.Unmarshal([]byte(m.AllowedDomains), &allowedDomains); err != nil {
			return nil, fmt.Errorf("unmarshal allowed_domains: %w", err)
		}
	}
	if allowedDomains == nil {
		allowedDomains = []string{}
	}

	var features map[string]bool
	if m.Features != "" {
		if err := json.Unmarshal([]byte(m.Features), &features); err != nil {
			return nil, fmt.Errorf("unmarshal features: %w", err)
		}
	}
	if features == nil {
		features = map[string]bool{}
	}

	return &domain.Tenant{
		ID:             m.ID,
		Name:           m.Name,
		Slug:           m.Slug,
		Plan:           domain.TenantPlan(m.Plan),
		Status:         domain.TenantStatus(m.Status),
		MaxUsers:       m.MaxUsers,
		AllowedDomains: allowedDomains,
		Settings: domain.TenantSettings{
			MaxUsers:       m.MaxUsers,
			AllowedDomains: allowedDomains,
			Features:       features,
			NotificationPrefs: domain.NotificationPreferences{
				EmailEnabled: true,
				InAppEnabled: true,
			},
		},
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}
