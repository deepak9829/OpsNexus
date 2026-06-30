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

// organizationModel is the GORM representation of the organizations table.
type organizationModel struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	TenantID  string    `gorm:"type:varchar(36);not null;index:idx_org_tenant_id"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Type      string    `gorm:"type:varchar(100)"`
	ParentID  *string   `gorm:"type:varchar(36)"`
	Metadata  string    `gorm:"type:text"` // JSON-encoded map[string]any
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (organizationModel) TableName() string { return "organizations" }

// organizationRepository implements ports.OrganizationRepository.
type organizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository constructs a GORM-backed OrganizationRepository.
func NewOrganizationRepository(db *gorm.DB) ports.OrganizationRepository {
	return &organizationRepository{db: db}
}

func (r *organizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	m, err := domainToOrgModel(org)
	if err != nil {
		return err
	}
	if res := r.db.WithContext(ctx).Create(m); res.Error != nil {
		return fmt.Errorf("insert organization: %w", res.Error)
	}
	return nil
}

func (r *organizationRepository) FindByID(ctx context.Context, id string) (*domain.Organization, error) {
	var m organizationModel
	res := r.db.WithContext(ctx).First(&m, "id = ?", id)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("find org by id: %w", res.Error)
	}
	return orgModelToDomain(&m)
}

func (r *organizationRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Organization, error) {
	var models []organizationModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list orgs by tenant: %w", err)
	}

	orgs := make([]*domain.Organization, 0, len(models))
	for i := range models {
		o, err := orgModelToDomain(&models[i])
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, nil
}

func (r *organizationRepository) Update(ctx context.Context, org *domain.Organization) error {
	m, err := domainToOrgModel(org)
	if err != nil {
		return err
	}
	if res := r.db.WithContext(ctx).Save(m); res.Error != nil {
		return fmt.Errorf("update organization: %w", res.Error)
	}
	return nil
}

func (r *organizationRepository) Delete(ctx context.Context, id string) error {
	if res := r.db.WithContext(ctx).Delete(&organizationModel{}, "id = ?", id); res.Error != nil {
		return fmt.Errorf("delete organization: %w", res.Error)
	}
	return nil
}

// ─── mapping helpers ──────────────────────────────────────────────────────────

func domainToOrgModel(o *domain.Organization) (*organizationModel, error) {
	meta := o.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	return &organizationModel{
		ID:        o.ID,
		TenantID:  o.TenantID,
		Name:      o.Name,
		Type:      o.Type,
		ParentID:  o.ParentID,
		Metadata:  string(metaJSON),
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}, nil
}

func orgModelToDomain(m *organizationModel) (*domain.Organization, error) {
	var metadata map[string]any
	if m.Metadata != "" {
		if err := json.Unmarshal([]byte(m.Metadata), &metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}

	return &domain.Organization{
		ID:        m.ID,
		TenantID:  m.TenantID,
		Name:      m.Name,
		Type:      m.Type,
		ParentID:  m.ParentID,
		Metadata:  metadata,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}
