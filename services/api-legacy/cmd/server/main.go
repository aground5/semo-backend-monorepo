// File: /Users/k2zoo/Documents/growingup/ox-hr/authn/cmd/main.go
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"semo-server/configs"
	httpEngine "semo-server/internal/app/http"
	"semo-server/internal/repositories"

	"github.com/wekeepgrowing/semo-backend-monorepo/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "", "Path to config file (short)")
	flag.StringVar(&configPath, "config", "", "Path to config file (long)")
	flag.Parse()

	// Set CONFIG_PATH environment variable if config path is provided
	if configPath != "" {
		os.Setenv("CONFIG_PATH", configPath)
	}

	// Initialize configuration
	configs.Init(&configPath)

	logConfig := logger.Config{
		Level: configs.Configs.Logs.LogLevel,
		Format: "json",
		Development: false,
	}

	// 로그 출력 설정
	if configs.Configs.Logs.StdoutOnly {
		logConfig.Output = "stdout"
	} else {
		logConfig.Output = "file"
		logConfig.FilePath = configs.Configs.Logs.LogPath
	}

	// 로거 생성
	log, err := logger.NewZapLogger(logConfig)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	
	// 애플리케이션 종료 시 로그 버퍼 정리
	defer log.Sync()

	// 로깅 시작
	log.Info("Configuration loaded.",
		zap.String("configPath", configPath),
	)

	// Initialize repositories (Postgres, Redis)
	repositories.Init(log)

	// Create gRPC server and run it in a separate goroutine.
	//grpcServer := grpcEngine.NewGRPCServer()
	//go grpcServer.Start()

	// Create HTTP server and run it in a separate goroutine.
	httpServer := httpEngine.NewServer(log)
	go func() {
		if err := httpServer.Start(); err != nil {
			// http.ErrServerClosed는 정상 종료 시 반환하는 에러입니다.
			if err.Error() != "http: Server closed" {
				log.Fatal("HTTP server error", zap.Error(err))
			}
		}
	}()

	// Graceful shutdown: OS 신호(예, SIGINT, SIGTERM)를 기다림
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("Shutdown signal received")

	// graceful shutdown을 위한 타임아웃 컨텍스트 생성 (예: 10초)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// HTTP 서버 graceful shutdown
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		log.Info("HTTP server shutdown gracefully")
	}

	// gRPC 서버 graceful shutdown
	//grpcServer.Shutdown()
	//configs.Logger.Info("gRPC server shutdown gracefully")

	log.Info("Server exited")
}
