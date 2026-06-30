package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/opsnexus/workflow-service/internal/domain"
	"github.com/opsnexus/workflow-service/internal/ports"
	"gorm.io/gorm"
)

type taskModel struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)"`
	CaseID      string     `gorm:"type:varchar(36);index"`
	TenantID    string     `gorm:"type:varchar(36);index"`
	Title       string     `gorm:"type:varchar(500)"`
	Description string     `gorm:"type:text"`
	Status      string     `gorm:"type:varchar(30);index"`
	AssigneeID  *string    `gorm:"type:varchar(36)"`
	DueAt       *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (taskModel) TableName() string { return "tasks" }

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) ports.TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *domain.Task) error {
	m := domainTaskToModel(task)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *taskRepository) FindByID(ctx context.Context, id string) (*domain.Task, error) {
	var m taskModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, err
	}
	return modelToDomainTask(&m), nil
}

func (r *taskRepository) ListByCase(ctx context.Context, caseID string) ([]*domain.Task, error) {
	var models []taskModel
	if err := r.db.WithContext(ctx).Where("case_id = ?", caseID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	tasks := make([]*domain.Task, 0, len(models))
	for i := range models {
		tasks = append(tasks, modelToDomainTask(&models[i]))
	}
	return tasks, nil
}

func (r *taskRepository) Update(ctx context.Context, task *domain.Task) error {
	m := domainTaskToModel(task)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *taskRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&taskModel{}, "id = ?", id).Error
}

func domainTaskToModel(t *domain.Task) *taskModel {
	return &taskModel{
		ID:          t.ID,
		CaseID:      t.CaseID,
		TenantID:    t.TenantID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		AssigneeID:  t.AssigneeID,
		DueAt:       t.DueAt,
		CompletedAt: t.CompletedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func modelToDomainTask(m *taskModel) *domain.Task {
	return &domain.Task{
		ID:          m.ID,
		CaseID:      m.CaseID,
		TenantID:    m.TenantID,
		Title:       m.Title,
		Description: m.Description,
		Status:      domain.TaskStatus(m.Status),
		AssigneeID:  m.AssigneeID,
		DueAt:       m.DueAt,
		CompletedAt: m.CompletedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
