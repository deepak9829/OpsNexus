package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/opsnexus/auth-service/internal/domain"
	"github.com/opsnexus/auth-service/internal/ports"
	"gorm.io/gorm"
)

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) ports.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(ctx context.Context, role *domain.Role) error {
	model := fromDomainRole(role)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("creating role: %w", err)
	}
	return nil
}

func (r *roleRepository) FindByID(ctx context.Context, id string) (*domain.Role, error) {
	var model roleModel
	err := r.db.WithContext(ctx).
		Preload("Permissions").
		First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRoleNotFound
		}
		return nil, fmt.Errorf("finding role by id: %w", err)
	}
	role := toDomainRole(&model)
	return &role, nil
}

func (r *roleRepository) FindByName(ctx context.Context, tenantID, name string) (*domain.Role, error) {
	var model roleModel
	err := r.db.WithContext(ctx).
		Preload("Permissions").
		First(&model, "tenant_id = ? AND name = ?", tenantID, name).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRoleNotFound
		}
		return nil, fmt.Errorf("finding role by name: %w", err)
	}
	role := toDomainRole(&model)
	return &role, nil
}

func (r *roleRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Role, error) {
	var models []roleModel
	if err := r.db.WithContext(ctx).
		Preload("Permissions").
		Where("tenant_id = ?", tenantID).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}

	roles := make([]*domain.Role, 0, len(models))
	for i := range models {
		role := toDomainRole(&models[i])
		roles = append(roles, &role)
	}
	return roles, nil
}

func (r *roleRepository) AssignToUser(ctx context.Context, userID, roleID string) error {
	var user userModel
	if err := r.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return domain.ErrUserNotFound
	}
	var role roleModel
	if err := r.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		return domain.ErrRoleNotFound
	}
	if err := r.db.WithContext(ctx).Model(&user).Association("Roles").Append(&role); err != nil {
		return fmt.Errorf("assigning role: %w", err)
	}
	return nil
}

func (r *roleRepository) RevokeFromUser(ctx context.Context, userID, roleID string) error {
	var user userModel
	if err := r.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return domain.ErrUserNotFound
	}
	var role roleModel
	if err := r.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		return domain.ErrRoleNotFound
	}
	if err := r.db.WithContext(ctx).Model(&user).Association("Roles").Delete(&role); err != nil {
		return fmt.Errorf("revoking role: %w", err)
	}
	return nil
}

func fromDomainRole(r *domain.Role) roleModel {
	perms := make([]permissionModel, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, permissionModel{
			ID:       p.ID,
			Resource: p.Resource,
			Action:   p.Action,
		})
	}
	return roleModel{
		ID:          r.ID,
		TenantID:    r.TenantID,
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Permissions: perms,
	}
}
