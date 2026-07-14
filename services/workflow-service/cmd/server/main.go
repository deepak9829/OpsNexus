package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/opsnexus/workflow-service/internal/adapters/mysql"
	httpAdapter "github.com/opsnexus/workflow-service/internal/adapters/http"
	"github.com/opsnexus/workflow-service/internal/application"
	"github.com/opsnexus/workflow-service/internal/config"
	"go.uber.org/zap"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	// Logger
	var logger *zap.Logger
	if cfg.Log.JSON {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	dbCfg := mysql.Config{
		DSN:             cfg.Database.DSN,
		DBHost:          cfg.Database.Host,
		DBPort:          cfg.Database.Port,
		DBUser:          cfg.Database.User,
		DBPassword:      cfg.Database.Password,
		DBName:          cfg.Database.Name,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := mysql.EnsureDatabase(dbCfg); err != nil {
			logger.Fatal("failed to ensure database", zap.Error(err))
		}
		db, err := mysql.NewDB(dbCfg)
		if err != nil {
			logger.Fatal("connecting to database", zap.Error(err))
		}
		if err := mysql.AutoMigrate(db); err != nil {
			logger.Fatal("running migrations", zap.Error(err))
		}
		logger.Info("migrations complete", zap.String("database", cfg.Database.Name))
		return
	}

	// Database
	db, err := mysql.NewDB(dbCfg)
	if err != nil {
		logger.Fatal("connecting to database", zap.Error(err))
	}
	logger.Info("database connected")

	// Migrations
	if err := mysql.AutoMigrate(db); err != nil {
		logger.Fatal("running migrations", zap.Error(err))
	}
	logger.Info("migrations applied")

	// Repositories
	caseRepo := mysql.NewCaseRepository(db)
	taskRepo := mysql.NewTaskRepository(db)
	wfRepo := mysql.NewWorkflowTemplateRepository(db)
	commentRepo := mysql.NewCommentRepository(db)

	// Services
	caseService := application.NewCaseService(caseRepo, wfRepo, commentRepo, logger)
	taskService := application.NewTaskService(taskRepo, caseRepo, logger)
	workflowService := application.NewWorkflowService(wfRepo, logger)

	// HTTP Handler & Router
	handler := httpAdapter.NewHandler(caseService, taskService, workflowService, logger)
	router := httpAdapter.NewRouter(handler, logger)

	// HTTP Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting workflow service", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for signal or error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		logger.Fatal("server error", zap.Error(err))
	case sig := <-quit:
		logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	} else {
		logger.Info("server stopped gracefully")
	}
}
