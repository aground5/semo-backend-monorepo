package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/grpc"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/http"
	"go.uber.org/zap"
)

func main() {
	// 설정 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("설정 로드 실패: %v", err)
	}

	// 로거 가져오기
	logger := cfg.Logger
	defer logger.Sync()

	logger.Info("인증 서비스를 시작합니다...",
		zap.String("service", cfg.Service.Name),
		zap.String("version", cfg.Service.Version),
	)

	// 인프라스트럭처 초기화
	infrastructure, err := db.NewInfrastructure(cfg)
	if err != nil {
		logger.Fatal("인프라스트럭처 초기화 실패", zap.Error(err))
	}
	defer infrastructure.Close()

	// HTTP 서버 설정
	httpConfig := http.Config{
		Port:    cfg.Server.HTTP.Port,
		Timeout: cfg.Server.HTTP.Timeout,
		Debug:   cfg.Server.HTTP.Debug,
	}

	// HTTP 서버 생성
	httpServer := http.NewServer(httpConfig, logger)
	httpServer.RegisterRoutes()

	// gRPC 서버 설정
	grpcConfig := grpc.Config{
		Port:    cfg.Server.GRPC.Port,
		Timeout: cfg.Server.GRPC.Timeout,
	}

	// gRPC 서버 생성
	grpcServer := grpc.NewServer(grpcConfig, logger)
	grpcServer.RegisterServices()

	// 서버 시작
	go func() {
		if err := httpServer.Start(); err != nil {
			logger.Error("HTTP 서버 종료", zap.Error(err))
		}
	}()

	go func() {
		if err := grpcServer.Start(); err != nil {
			logger.Error("gRPC 서버 종료", zap.Error(err))
		}
	}()

	// 그레이스풀 종료를 위한 시그널 처리
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("서버를 종료합니다...")

	// HTTP 서버 종료
	if err := httpServer.Stop(); err != nil {
		logger.Error("HTTP 서버 종료 오류", zap.Error(err))
	}

	// gRPC 서버 종료
	grpcServer.Stop()

	logger.Info("서버가 정상적으로 종료되었습니다")
}
