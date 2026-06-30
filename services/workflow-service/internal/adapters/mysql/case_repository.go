package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opsnexus/workflow-service/internal/domain"
	"github.com/opsnexus/workflow-service/internal/ports"
	"gorm.io/gorm"
)

type caseModel struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)"`
	TenantID    string     `gorm:"type:varchar(36);index"`
	CaseNumber  string     `gorm:"type:varchar(50);index"`
	Title       string     `gorm:"type:varchar(500)"`
	Description string     `gorm:"type:text"`
	Status      string     `gorm:"type:varchar(30);index"`
	Priority    string     `gorm:"type:varchar(20);index"`
	AssigneeID  *string    `gorm:"type:varchar(36)"`
	ReporterID  string     `gorm:"type:varchar(36);index"`
	WorkflowID  *string    `gorm:"type:varchar(36)"`
	SLADueAt    *time.Time
	SLABreached bool       `gorm:"default:false"`
	Tags        string     `gorm:"type:text"`
	ResolvedAt  *time.Time
	ClosedAt    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (caseModel) TableName() string { return "cases" }

type caseTransitionModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"`
	CaseID      string    `gorm:"type:varchar(36);index"`
	FromStatus  string    `gorm:"type:varchar(30)"`
	ToStatus    string    `gorm:"type:varchar(30)"`
	Reason      string    `gorm:"type:varchar(500)"`
	PerformedBy string    `gorm:"type:varchar(36)"`
	PerformedAt time.Time
}

func (caseTransitionModel) TableName() string { return "case_transitions" }

type caseCounterModel struct {
	TenantID string `gorm:"primaryKey;type:varchar(36)"`
	Counter  int64  `gorm:"default:0"`
}

func (caseCounterModel) TableName() string { return "case_counters" }

type caseRepository struct {
	db *gorm.DB
}

func NewCaseRepository(db *gorm.DB) ports.CaseRepository {
	return &caseRepository{db: db}
}

func (r *caseRepository) Create(ctx context.Context, c *domain.Case) error {
	m, err := domainCaseToModel(c)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *caseRepository) FindByID(ctx context.Context, id string) (*domain.Case, error) {
	var m caseModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCaseNotFound
		}
		return nil, err
	}
	return modelToDomainCase(&m)
}

func (r *caseRepository) FindByNumber(ctx context.Context, tenantID, caseNumber string) (*domain.Case, error) {
	var m caseModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND case_number = ?", tenantID, caseNumber).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCaseNotFound
		}
		return nil, err
	}
	return modelToDomainCase(&m)
}

func (r *caseRepository) Update(ctx context.Context, c *domain.Case) error {
	m, err := domainCaseToModel(c)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *caseRepository) List(ctx context.Context, tenantID string, filter ports.CaseFilter, page, limit int) ([]*domain.Case, int64, error) {
	q := r.db.WithContext(ctx).Model(&caseModel{}).Where("tenant_id = ?", tenantID)

	if filter.Status != nil {
		q = q.Where("status = ?", string(*filter.Status))
	}
	if filter.Priority != nil {
		q = q.Where("priority = ?", string(*filter.Priority))
	}
	if filter.AssigneeID != nil {
		q = q.Where("assignee_id = ?", *filter.AssigneeID)
	}
	if filter.ReporterID != nil {
		q = q.Where("reporter_id = ?", *filter.ReporterID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	var models []caseModel
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	cases := make([]*domain.Case, 0, len(models))
	for i := range models {
		c, err := modelToDomainCase(&models[i])
		if err != nil {
			return nil, 0, err
		}
		cases = append(cases, c)
	}
	return cases, total, nil
}

func (r *caseRepository) RecordTransition(ctx context.Context, t *domain.CaseTransition) error {
	m := &caseTransitionModel{
		ID:          t.ID,
		CaseID:      t.CaseID,
		FromStatus:  string(t.FromStatus),
		ToStatus:    string(t.ToStatus),
		Reason:      t.Reason,
		PerformedBy: t.PerformedBy,
		PerformedAt: t.PerformedAt,
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *caseRepository) GetTransitionHistory(ctx context.Context, caseID string) ([]*domain.CaseTransition, error) {
	var models []caseTransitionModel
	if err := r.db.WithContext(ctx).Where("case_id = ?", caseID).Order("performed_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.CaseTransition, 0, len(models))
	for _, m := range models {
		result = append(result, &domain.CaseTransition{
			ID:          m.ID,
			CaseID:      m.CaseID,
			FromStatus:  domain.CaseStatus(m.FromStatus),
			ToStatus:    domain.CaseStatus(m.ToStatus),
			Reason:      m.Reason,
			PerformedBy: m.PerformedBy,
			PerformedAt: m.PerformedAt,
		})
	}
	return result, nil
}

func (r *caseRepository) NextCaseNumber(ctx context.Context, tenantID string) (string, error) {
	var counter int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m caseCounterModel
		result := tx.Where("tenant_id = ?", tenantID).First(&m)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			m = caseCounterModel{TenantID: tenantID, Counter: 1}
			if err := tx.Create(&m).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		} else {
			m.Counter++
			if err := tx.Save(&m).Error; err != nil {
				return err
			}
		}
		counter = m.Counter
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("incrementing case counter: %w", err)
	}

	// Format: CASE-{tenantID prefix 5 chars}-{seq 05d}
	prefix := tenantID
	if len(prefix) > 5 {
		prefix = strings.ToUpper(prefix[:5])
	} else {
		prefix = strings.ToUpper(prefix)
	}
	return fmt.Sprintf("CASE-%s-%05d", prefix, counter), nil
}

// Conversion helpers

func domainCaseToModel(c *domain.Case) (*caseModel, error) {
	tagsJSON, err := json.Marshal(c.Tags)
	if err != nil {
		return nil, fmt.Errorf("marshaling tags: %w", err)
	}
	return &caseModel{
		ID:          c.ID,
		TenantID:    c.TenantID,
		CaseNumber:  c.CaseNumber,
		Title:       c.Title,
		Description: c.Description,
		Status:      string(c.Status),
		Priority:    string(c.Priority),
		AssigneeID:  c.AssigneeID,
		ReporterID:  c.ReporterID,
		WorkflowID:  c.WorkflowID,
		SLADueAt:    c.SLA.DueAt,
		SLABreached: c.SLA.Breached,
		Tags:        string(tagsJSON),
		ResolvedAt:  c.ResolvedAt,
		ClosedAt:    c.ClosedAt,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}, nil
}

func modelToDomainCase(m *caseModel) (*domain.Case, error) {
	var tags []string
	if m.Tags != "" {
		if err := json.Unmarshal([]byte(m.Tags), &tags); err != nil {
			tags = []string{}
		}
	}
	return &domain.Case{
		ID:          m.ID,
		TenantID:    m.TenantID,
		CaseNumber:  m.CaseNumber,
		Title:       m.Title,
		Description: m.Description,
		Status:      domain.CaseStatus(m.Status),
		Priority:    domain.CasePriority(m.Priority),
		AssigneeID:  m.AssigneeID,
		ReporterID:  m.ReporterID,
		WorkflowID:  m.WorkflowID,
		SLA: domain.SLA{
			DueAt:    m.SLADueAt,
			Breached: m.SLABreached,
		},
		Tags:       tags,
		ResolvedAt: m.ResolvedAt,
		ClosedAt:   m.ClosedAt,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}, nil
}
