package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/infrastructure/db"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/init"
)

func main() {
	// 애플리케이션 초기화
	if err := init.Init(); err != nil {
		log.Fatalf("애플리케이션 초기화 실패: %v", err)
	}

	// 설정 로드
	cfg := config.AppConfig
	logger := cfg.Logger

	logger.Info("API 서비스 시작 중...")

	// 데이터베이스 연결
	database, err := db.NewDatabaseConnection(cfg)
	if err != nil {
		logger.Fatal("데이터베이스 연결 실패", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		logger.Fatal("SQL DB 인스턴스 가져오기 실패", err)
	}
	defer sqlDB.Close()

	// TODO: 레포지토리, 유스케이스, 핸들러 초기화

	// TODO: HTTP 서버 설정 및 시작
	// httpServer := http.NewServer(cfg)
	// router := httpServer.Router()
	// 라우팅 설정...

	// TODO: gRPC 서버 설정 및 시작
	// grpcServer := grpc.NewServer(cfg)
	// 서비스 등록...

	// 서버 시작
	// go func() {
	// 	logger.Info("HTTP 서버 시작", "port", cfg.Server.HTTP.Port)
	// 	if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
	// 		logger.Fatal("HTTP 서버 시작 실패", err)
	// 	}
	// }()

	// go func() {
	// 	logger.Info("gRPC 서버 시작", "port", cfg.Server.GRPC.Port)
	// 	if err := grpcServer.Start(); err != nil {
	// 		logger.Fatal("gRPC 서버 시작 실패", err)
	// 	}
	// }()

	// 종료 신호 처리
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("서버 종료 중...")

	// 최대 30초 동안 남은 요청 처리
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// HTTP/gRPC 서버 종료
	// if err := httpServer.Shutdown(ctx); err != nil {
	// 	logger.Error("HTTP 서버 종료 오류", err)
	// }
	// grpcServer.Shutdown()

	logger.Info("서버 정상 종료")
}
