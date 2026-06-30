package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "github.com/opsnexus/document-service/internal/adapters/http"
	"github.com/opsnexus/document-service/internal/adapters/mongodb"
	"github.com/opsnexus/document-service/internal/application"
	"github.com/opsnexus/document-service/internal/config"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	db, err := mongodb.NewClient(cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		logger.Fatal("failed to connect to MongoDB", zap.Error(err))
	}

	ctx := context.Background()

	formRepo := mongodb.NewFormTemplateRepository(db)
	submissionRepo := mongodb.NewFormSubmissionRepository(db)
	docRepo := mongodb.NewDocumentRepository(db)

	type indexer interface {
		EnsureIndexes(context.Context) error
	}

	for _, repo := range []any{formRepo, submissionRepo, docRepo} {
		if r, ok := repo.(indexer); ok {
			if err := r.EnsureIndexes(ctx); err != nil {
				logger.Warn("failed to ensure indexes", zap.Error(err))
			}
		}
	}

	formSvc := application.NewFormService(formRepo, submissionRepo, logger)
	docSvc := application.NewDocumentService(docRepo, cfg.UploadDir, logger)

	handler := httpAdapter.NewHandler(formSvc, docSvc, logger)
	router := httpAdapter.NewRouter(handler, logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("document service starting", zap.String("port", cfg.Port))
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
