package mysql

import (
	"fmt"

	"github.com/opsnexus/auth-service/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// EnsureDatabase connects without a database name and creates the target
// database if it does not already exist. Safe to call on every startup.
func EnsureDatabase(cfg config.DatabaseConfig) error {
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=UTC",
		cfg.User, cfg.Password, cfg.Host, cfg.Port)
	db, err := gorm.Open(mysql.Open(rootDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("connecting to mysql (no db): %w", err)
	}
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Name)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("creating database %s: %w", cfg.Name, err)
	}
	return nil
}

func NewConnection(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	return db, nil
}
