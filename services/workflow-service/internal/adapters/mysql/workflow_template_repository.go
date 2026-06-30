package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/opsnexus/workflow-service/internal/domain"
	"github.com/opsnexus/workflow-service/internal/ports"
	"gorm.io/gorm"
)

type workflowTemplateModel struct {
	ID              string    `gorm:"primaryKey;type:varchar(36)"`
	TenantID        string    `gorm:"type:varchar(36);index"`
	Name            string    `gorm:"type:varchar(255)"`
	Description     string    `gorm:"type:text"`
	States          string    `gorm:"type:text"` // JSON
	Transitions     string    `gorm:"type:text"` // JSON
	DefaultPriority string    `gorm:"type:varchar(20)"`
	SLAHours        int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (workflowTemplateModel) TableName() string { return "workflow_templates" }

type workflowTemplateRepository struct {
	db *gorm.DB
}

func NewWorkflowTemplateRepository(db *gorm.DB) ports.WorkflowTemplateRepository {
	return &workflowTemplateRepository{db: db}
}

func (r *workflowTemplateRepository) Create(ctx context.Context, w *domain.WorkflowTemplate) error {
	m, err := domainWFToModel(w)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *workflowTemplateRepository) FindByID(ctx context.Context, id string) (*domain.WorkflowTemplate, error) {
	var m workflowTemplateModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrWorkflowNotFound
		}
		return nil, err
	}
	return modelToDomainWF(&m)
}

func (r *workflowTemplateRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.WorkflowTemplate, error) {
	var models []workflowTemplateModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.WorkflowTemplate, 0, len(models))
	for i := range models {
		w, err := modelToDomainWF(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, nil
}

func (r *workflowTemplateRepository) Update(ctx context.Context, w *domain.WorkflowTemplate) error {
	m, err := domainWFToModel(w)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(m).Error
}

type transitionJSON struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

func domainWFToModel(w *domain.WorkflowTemplate) (*workflowTemplateModel, error) {
	statesJSON, err := json.Marshal(w.States)
	if err != nil {
		return nil, fmt.Errorf("marshaling states: %w", err)
	}

	tjs := make([]transitionJSON, len(w.Transitions))
	for i, t := range w.Transitions {
		tjs[i] = transitionJSON{From: string(t.From), To: string(t.To), Label: t.Label}
	}
	transitionsJSON, err := json.Marshal(tjs)
	if err != nil {
		return nil, fmt.Errorf("marshaling transitions: %w", err)
	}

	return &workflowTemplateModel{
		ID:              w.ID,
		TenantID:        w.TenantID,
		Name:            w.Name,
		Description:     w.Description,
		States:          string(statesJSON),
		Transitions:     string(transitionsJSON),
		DefaultPriority: string(w.DefaultPriority),
		SLAHours:        w.SLAHours,
		CreatedAt:       w.CreatedAt,
		UpdatedAt:       w.UpdatedAt,
	}, nil
}

func modelToDomainWF(m *workflowTemplateModel) (*domain.WorkflowTemplate, error) {
	var states []string
	if m.States != "" {
		if err := json.Unmarshal([]byte(m.States), &states); err != nil {
			states = []string{}
		}
	}

	var tjs []transitionJSON
	if m.Transitions != "" {
		if err := json.Unmarshal([]byte(m.Transitions), &tjs); err != nil {
			tjs = []transitionJSON{}
		}
	}
	transitions := make([]domain.Transition, len(tjs))
	for i, tj := range tjs {
		transitions[i] = domain.Transition{
			From:  domain.CaseStatus(tj.From),
			To:    domain.CaseStatus(tj.To),
			Label: tj.Label,
		}
	}

	return &domain.WorkflowTemplate{
		ID:              m.ID,
		TenantID:        m.TenantID,
		Name:            m.Name,
		Description:     m.Description,
		States:          states,
		Transitions:     transitions,
		DefaultPriority: domain.CasePriority(m.DefaultPriority),
		SLAHours:        m.SLAHours,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}, nil
}
