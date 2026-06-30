package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/opsnexus/workflow-service/internal/domain"
	"github.com/opsnexus/workflow-service/internal/ports"
	"gorm.io/gorm"
)

type commentModel struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	CaseID    string    `gorm:"type:varchar(36);index"`
	TenantID  string    `gorm:"type:varchar(36);index"`
	AuthorID  string    `gorm:"type:varchar(36);index"`
	Body      string    `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (commentModel) TableName() string { return "comments" }

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) ports.CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(ctx context.Context, c *domain.Comment) error {
	m := &commentModel{
		ID:        c.ID,
		CaseID:    c.CaseID,
		TenantID:  c.TenantID,
		AuthorID:  c.AuthorID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *commentRepository) FindByID(ctx context.Context, id string) (*domain.Comment, error) {
	var m commentModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCommentNotFound
		}
		return nil, err
	}
	return &domain.Comment{
		ID:        m.ID,
		CaseID:    m.CaseID,
		TenantID:  m.TenantID,
		AuthorID:  m.AuthorID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *commentRepository) ListByCase(ctx context.Context, caseID string) ([]*domain.Comment, error) {
	var models []commentModel
	if err := r.db.WithContext(ctx).Where("case_id = ?", caseID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.Comment, 0, len(models))
	for _, m := range models {
		result = append(result, &domain.Comment{
			ID:        m.ID,
			CaseID:    m.CaseID,
			TenantID:  m.TenantID,
			AuthorID:  m.AuthorID,
			Body:      m.Body,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		})
	}
	return result, nil
}

func (r *commentRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&commentModel{}, "id = ?", id).Error
}
