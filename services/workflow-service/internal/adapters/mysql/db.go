package mysql

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DSN             string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// EnsureDatabase connects without a database name and creates the target
// database if it does not already exist. Safe to call on every startup.
func EnsureDatabase(cfg Config) error {
	host := cfg.DBHost
	if host == "" {
		host = "localhost"
	}
	port := cfg.DBPort
	if port == "" {
		port = "3306"
	}
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=UTC",
		cfg.DBUser, cfg.DBPassword, host, port)
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

func NewDB(cfg Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("opening mysql connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("getting sql.DB: %w", err)
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 25
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 10
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime == 0 {
		lifetime = 5 * time.Minute
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(lifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("pinging mysql: %w", err)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&caseModel{},
		&caseTransitionModel{},
		&caseCounterModel{},
		&taskModel{},
		&workflowTemplateModel{},
		&commentModel{},
	)
}
