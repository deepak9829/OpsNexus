package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	httpAdapter "github.com/opsnexus/tenant-service/internal/adapters/http"
	"github.com/opsnexus/tenant-service/internal/adapters/mysql"
	"github.com/opsnexus/tenant-service/internal/application"
	"github.com/opsnexus/tenant-service/internal/config"
)

func main() {
	// ── Logger ────────────────────────────────────────────────────────────────
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := mysql.EnsureDatabase(cfg); err != nil {
			logger.Fatal("failed to ensure database", zap.Error(err))
		}
		db, err := mysql.NewDB(cfg)
		if err != nil {
			logger.Fatal("failed to connect to database", zap.Error(err))
		}
		if err := mysql.AutoMigrate(db); err != nil {
			logger.Fatal("failed to run migrations", zap.Error(err))
		}
		logger.Info("migrations complete", zap.String("database", cfg.DBName))
		return
	}

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := mysql.NewDB(cfg)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	// ── Migrations ────────────────────────────────────────────────────────────
	if err := mysql.AutoMigrate(db); err != nil {
		logger.Fatal("running migrations", zap.Error(err))
	}
	logger.Info("migrations applied")

	// ── Repositories ──────────────────────────────────────────────────────────
	tenantRepo := mysql.NewTenantRepository(db)
	orgRepo := mysql.NewOrganizationRepository(db)
	profileRepo := mysql.NewUserProfileRepository(db)

	// ── Services ──────────────────────────────────────────────────────────────
	tenantSvc := application.NewTenantService(tenantRepo, logger)
	orgSvc := application.NewOrganizationService(orgRepo, tenantRepo, logger)
	profileSvc := application.NewUserProfileService(profileRepo, tenantRepo, logger)

	// ── HTTP ──────────────────────────────────────────────────────────────────
	handler := httpAdapter.NewHandler(tenantSvc, orgSvc, profileSvc, logger)
	router := httpAdapter.NewRouter(handler, logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Start ─────────────────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("tenant-service starting", zap.Int("port", cfg.Port), zap.String("env", cfg.ServerEnv))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server listen error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down server…")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}

	logger.Info("server stopped")
}
