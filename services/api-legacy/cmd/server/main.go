// File: /Users/k2zoo/Documents/growingup/ox-hr/authn/cmd/main.go
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"semo-server/configs"
	httpEngine "semo-server/internal/app/http"
	"semo-server/internal/repositories"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "", "Path to config file (short)")
	flag.StringVar(&configPath, "config", "", "Path to config file (long)")
	flag.Parse()

	// Initialize configuration
	configs.Init(&configPath)
	configs.Logger.Info("Configuration loaded.",
		zap.String("configPath", configPath),
	)

	// Initialize repositories (Postgres, Redis)
	repositories.Init()

	// Create gRPC server and run it in a separate goroutine.
	//grpcServer := grpcEngine.NewGRPCServer()
	//go grpcServer.Start()

	// Create HTTP server and run it in a separate goroutine.
	httpServer := httpEngine.NewServer()
	go func() {
		if err := httpServer.Start(); err != nil {
			// http.ErrServerClosed는 정상 종료 시 반환하는 에러입니다.
			if err.Error() != "http: Server closed" {
				configs.Logger.Fatal("HTTP server error", zap.Error(err))
			}
		}
	}()

	// Graceful shutdown: OS 신호(예, SIGINT, SIGTERM)를 기다림
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	configs.Logger.Info("Shutdown signal received")

	// graceful shutdown을 위한 타임아웃 컨텍스트 생성 (예: 10초)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// HTTP 서버 graceful shutdown
	if err := httpServer.Shutdown(ctx); err != nil {
		configs.Logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		configs.Logger.Info("HTTP server shutdown gracefully")
	}

	// gRPC 서버 graceful shutdown
	//grpcServer.Shutdown()
	//configs.Logger.Info("gRPC server shutdown gracefully")

	configs.Logger.Info("Server exited")
}
