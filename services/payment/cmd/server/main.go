package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/database"
	grpcServer "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/grpc"
	httpServer "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/http"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize database connection
	db, err := database.NewConnection(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := database.Close(db, logger); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		}
	}()

	// Run database migrations
	if err := database.Migrate(db, logger); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	// Initialize repositories
	repos := database.NewRepositories(db, &cfg.Service.Supabase, logger)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize servers
	grpcSrv := grpcServer.NewServer(cfg, logger)
	httpSrv := httpServer.NewServer(cfg, logger, repos)

	// Start servers
	go func() {
		if err := grpcSrv.Start(); err != nil {
			logger.Fatal("Failed to start gRPC server", zap.Error(err))
		}
	}()

	go func() {
		if err := httpSrv.Start(); err != nil {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down servers...")

	// Shutdown servers
	if err := grpcSrv.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown gRPC server", zap.Error(err))
	}

	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown HTTP server", zap.Error(err))
	}

	logger.Info("Servers shut down successfully")
}
