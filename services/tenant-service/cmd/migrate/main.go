package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/opsnexus/tenant-service/internal/adapters/mysql"
	"github.com/opsnexus/tenant-service/internal/config"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}

	db, err := mysql.NewDB(cfg)
	if err != nil {
		logger.Fatal("connect to database", zap.Error(err))
	}

	logger.Info("running auto-migrate…")

	if err := db.AutoMigrate(
		&mysql.TenantModelExported{},
		&mysql.OrganizationModelExported{},
		&mysql.UserProfileModelExported{},
	); err != nil {
		fmt.Fprintf(os.Stderr, "auto-migrate failed: %v\n", err)
		os.Exit(1)
	}

	logger.Info("migration completed successfully")
}
