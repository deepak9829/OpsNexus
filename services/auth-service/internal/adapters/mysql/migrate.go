package mysql

import (
	"fmt"

	"gorm.io/gorm"
)

// AutoMigrate runs GORM AutoMigrate for all models in the mysql package.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&userModel{},
		&roleModel{},
		&permissionModel{},
		&refreshTokenModel{},
	); err != nil {
		return fmt.Errorf("auto migrating schema: %w", err)
	}
	return nil
}
