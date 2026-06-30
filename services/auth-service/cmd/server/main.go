package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "github.com/opsnexus/auth-service/internal/adapters/http"
	mysqladapter "github.com/opsnexus/auth-service/internal/adapters/mysql"
	"github.com/opsnexus/auth-service/internal/application"
	"github.com/opsnexus/auth-service/internal/config"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	var logger *zap.Logger
	if cfg.Server.Env == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck // best-effort flush

	db, err := mysqladapter.NewConnection(cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	if err := mysqladapter.AutoMigrate(db); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	userRepo := mysqladapter.NewUserRepository(db)
	roleRepo := mysqladapter.NewRoleRepository(db)
	tokenRepo := mysqladapter.NewRefreshTokenRepository(db)

	authSvc := application.NewAuthService(
		userRepo,
		roleRepo,
		tokenRepo,
		cfg.JWT.Secret,
		cfg.JWT.AccessTTLMinutes,
		cfg.JWT.RefreshTTLHours,
		logger,
	)

	handler := httpadapter.NewHandler(authSvc, logger)
	router := httpadapter.NewRouter(handler, authSvc, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("auth-service starting",
			zap.String("port", cfg.Server.Port),
			zap.String("env", cfg.Server.Env),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}
	logger.Info("server stopped")
}
