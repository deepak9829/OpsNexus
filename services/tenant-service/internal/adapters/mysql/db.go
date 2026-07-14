package mysql

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/opsnexus/tenant-service/internal/config"
)

// EnsureDatabase connects without a database name and creates the target
// database if it does not already exist. Safe to call on every startup.
func EnsureDatabase(cfg *config.Config) error {
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPass, cfg.MySQLHost, cfg.MySQLPort)
	db, err := gorm.Open(mysql.Open(rootDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("connecting to mysql (no db): %w", err)
	}
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.DBName)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("creating database %s: %w", cfg.DBName, err)
	}
	return nil
}

// AutoMigrate runs GORM schema migrations for all tenant-service models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&TenantModelExported{},
		&OrganizationModelExported{},
		&UserProfileModelExported{},
	)
}

// NewDB opens a GORM/MySQL connection using the supplied configuration.
func NewDB(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Silent
	if cfg.ServerEnv == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}
