package mysql

import "time"

// TenantModelExported is the exported GORM model used for AutoMigrate.
type TenantModelExported struct {
	ID             string    `gorm:"primaryKey;type:varchar(36)"`
	Name           string    `gorm:"type:varchar(255);not null"`
	Slug           string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Plan           string    `gorm:"type:varchar(50);not null;default:free"`
	Status         string    `gorm:"type:varchar(20);not null;default:active"`
	MaxUsers       int       `gorm:"not null;default:10"`
	AllowedDomains string    `gorm:"type:text"`
	Features       string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (TenantModelExported) TableName() string { return "tenants" }

// OrganizationModelExported is the exported GORM model used for AutoMigrate.
type OrganizationModelExported struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	TenantID  string    `gorm:"type:varchar(36);not null;index:idx_org_tenant_id"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Type      string    `gorm:"type:varchar(100)"`
	ParentID  *string   `gorm:"type:varchar(36)"`
	Metadata  string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (OrganizationModelExported) TableName() string { return "organizations" }

// UserProfileModelExported is the exported GORM model used for AutoMigrate.
type UserProfileModelExported struct {
	UserID         string    `gorm:"primaryKey;type:varchar(36)"`
	TenantID       string    `gorm:"type:varchar(36);not null;index:idx_profile_tenant_id"`
	OrganizationID *string   `gorm:"type:varchar(36)"`
	DisplayName    string    `gorm:"type:varchar(255)"`
	AvatarURL      string    `gorm:"type:varchar(512)"`
	Timezone       string    `gorm:"type:varchar(100);default:UTC"`
	Locale         string    `gorm:"type:varchar(20);default:en"`
	Metadata       string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (UserProfileModelExported) TableName() string { return "user_profiles" }
