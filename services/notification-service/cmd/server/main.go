package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	dynamoAdapter "github.com/opsnexus/notification-service/internal/adapters/dynamodb"
	httpAdapter "github.com/opsnexus/notification-service/internal/adapters/http"
	"github.com/opsnexus/notification-service/internal/application"
	"github.com/opsnexus/notification-service/internal/config"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	dynamoClient, err := dynamoAdapter.NewClient(
		cfg.AWSEndpointURL,
		cfg.AWSRegion,
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
	)
	if err != nil {
		logger.Fatal("failed to create DynamoDB client", zap.Error(err))
	}

	notifRepo := dynamoAdapter.NewNotificationRepository(dynamoClient, cfg.DynamoDBNotificationsTable)
	auditRepo := dynamoAdapter.NewAuditRepository(dynamoClient, cfg.DynamoDBAuditTable)

	notifSvc := application.NewNotificationService(notifRepo, logger)
	auditSvc := application.NewAuditService(auditRepo, logger)

	handler := httpAdapter.NewHandler(notifSvc, auditSvc, logger)
	router := httpAdapter.NewRouter(handler, logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("notification service starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}
	logger.Info("server exited")
}
