package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/opsnexus/auth-service/internal/domain"
	"github.com/opsnexus/auth-service/internal/ports"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) ports.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	model := fromDomainUser(user)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var model userModel
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Preload("Roles.Permissions").
		First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	return toDomainUser(&model), nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model userModel
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Preload("Roles.Permissions").
		First(&model, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("finding user by email: %w", err)
	}
	return toDomainUser(&model), nil
}

func (r *userRepository) FindByTenantAndEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	var model userModel
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Preload("Roles.Permissions").
		First(&model, "tenant_id = ? AND email = ?", tenantID, email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("finding user by tenant and email: %w", err)
	}
	return toDomainUser(&model), nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	model := fromDomainUser(user)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	return nil
}

func (r *userRepository) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.User, int64, error) {
	var models []userModel
	var count int64

	offset := (page - 1) * limit
	tx := r.db.WithContext(ctx).Model(&userModel{}).Where("tenant_id = ?", tenantID)

	if err := tx.Count(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("counting users: %w", err)
	}

	if err := tx.Preload("Roles").Preload("Roles.Permissions").
		Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("listing users: %w", err)
	}

	users := make([]*domain.User, 0, len(models))
	for i := range models {
		users = append(users, toDomainUser(&models[i]))
	}
	return users, count, nil
}

func toDomainUser(m *userModel) *domain.User {
	roles := make([]domain.Role, 0, len(m.Roles))
	for _, r := range m.Roles {
		roles = append(roles, toDomainRole(&r))
	}
	return &domain.User{
		ID:           m.ID,
		TenantID:     m.TenantID,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		FirstName:    m.FirstName,
		LastName:     m.LastName,
		Status:       domain.UserStatus(m.Status),
		Roles:        roles,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func fromDomainUser(u *domain.User) userModel {
	return userModel{
		ID:           u.ID,
		TenantID:     u.TenantID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		Status:       string(u.Status),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func toDomainRole(m *roleModel) domain.Role {
	perms := make([]domain.Permission, 0, len(m.Permissions))
	for _, p := range m.Permissions {
		perms = append(perms, domain.Permission{
			ID:       p.ID,
			Resource: p.Resource,
			Action:   p.Action,
		})
	}
	return domain.Role{
		ID:          m.ID,
		TenantID:    m.TenantID,
		Name:        m.Name,
		Description: m.Description,
		Permissions: perms,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
