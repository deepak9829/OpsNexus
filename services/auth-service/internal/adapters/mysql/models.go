package mysql

import "time"

type userModel struct {
	ID           string           `gorm:"primaryKey;type:varchar(36)"`
	TenantID     string           `gorm:"type:varchar(36);index"`
	Email        string           `gorm:"type:varchar(255);uniqueIndex"`
	PasswordHash string           `gorm:"type:varchar(255)"`
	FirstName    string           `gorm:"type:varchar(100)"`
	LastName     string           `gorm:"type:varchar(100)"`
	Status       string           `gorm:"type:varchar(20);default:active"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Roles        []roleModel      `gorm:"many2many:user_roles;"`
}

func (userModel) TableName() string { return "users" }

type roleModel struct {
	ID          string            `gorm:"primaryKey;type:varchar(36)"`
	TenantID    string            `gorm:"type:varchar(36)"`
	Name        string            `gorm:"type:varchar(100)"`
	Description string            `gorm:"type:varchar(255)"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Permissions []permissionModel `gorm:"many2many:role_permissions;"`
}

func (roleModel) TableName() string { return "roles" }

type permissionModel struct {
	ID       string `gorm:"primaryKey;type:varchar(36)"`
	Resource string `gorm:"type:varchar(100)"`
	Action   string `gorm:"type:varchar(50)"`
}

func (permissionModel) TableName() string { return "permissions" }

type refreshTokenModel struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	UserID    string    `gorm:"type:varchar(36);index"`
	TenantID  string    `gorm:"type:varchar(36)"`
	Token     string    `gorm:"type:varchar(512);uniqueIndex"`
	ExpiresAt time.Time
	Revoked   bool      `gorm:"default:false"`
	CreatedAt time.Time
}

func (refreshTokenModel) TableName() string { return "refresh_tokens" }
