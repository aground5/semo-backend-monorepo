package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// 데이터베이스 연결 설정
	dbConfig := db.Config{
		Driver:          cfg.Database.Driver,
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		Name:            cfg.Database.Name,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.Database.ConnMaxLifetime) * time.Second,
		SSLMode:         "disable", // 필요에 따라 변경
	}

	// 데이터베이스 연결
	_, err = db.NewPostgresDB(dbConfig, logger)
	if err != nil {
		logger.Fatal("데이터베이스 연결 실패", zap.Error(err))
	}

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
