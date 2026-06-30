package main

import (
	"fmt"
	"os"

	"github.com/opsnexus/workflow-service/internal/adapters/mysql"
	"github.com/opsnexus/workflow-service/internal/config"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	db, err := mysql.NewDB(mysql.Config{
		DSN:             cfg.Database.DSN,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		logger.Fatal("connecting to database", zap.Error(err))
	}

	logger.Info("running migrations...")
	if err := mysql.AutoMigrate(db); err != nil {
		logger.Fatal("migration failed", zap.Error(err))
	}
	logger.Info("migrations completed successfully")
}
