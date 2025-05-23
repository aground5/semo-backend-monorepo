package main

import (
	"context"
	// "flag" // 플래그 패키지는 더 이상 직접 사용하지 않을 수 있습니다.
	"os"
	"os/signal"
	"syscall"
	"time"

	// "semo-server/configs-legacy" // 기존 설정 패키지 제거
	"semo-server/config"
	httpEngine "semo-server/internal/app/http"
	"semo-server/internal/repositories"

	"github.com/wekeepgrowing/semo-backend-monorepo/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	// Initialize configuration using the new config package
	appCfg, err := config.Load() // 새 설정 로드 함수 호출
	if err != nil {
		// 초기 로거가 없을 수 있으므로 panic 또는 표준 로그 사용
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger with new configuration
	logConfig := logger.Config{
		Level:       appCfg.Log.Level, // 새 설정 값 사용
		Format:      appCfg.Log.Format, // 새 설정 값 사용 (기존 "json"과 일치 예상)
		Development: appCfg.Server.Debug, // 새 설정 값 사용
		Output:      appCfg.Log.Output,   // 새 설정 값 사용 (기존 StdoutOnly 로직 대체)
		// FilePath는 appCfg.Log.Output이 "file"일 경우 필요하며, yaml에 해당 경로 설정이 있어야 합니다.
		// 현재 api-legacy.yaml의 log.output이 stdout이므로 FilePath는 비워둡니다.
	}

	// 로거 생성
	log, err := logger.NewZapLogger(logConfig)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	
	// 애플리케이션 종료 시 로그 버퍼 정리
	defer func() {
		if err := log.Sync(); err != nil {
			// Sync 오류 처리 (예: 표준 오류로 출력)
			// fmt.Fprintf(os.Stderr, "Error syncing logger: %v\n", err)
		}
	}()

	// 로깅 시작
	log.Info("Configuration loaded successfully for api-legacy service.") // configPath 관련 로그 메시지 수정

	// Initialize repositories (Postgres, Redis)
	// repositories.Init는 내부적으로 apilegacyconfig.AppConfig를 사용하거나,
	// 필요한 설정값을 직접 전달받도록 수정될 수 있습니다.
	// 현재는 log 객체만 받으므로 그대로 둡니다.
	repositories.Init(log, appCfg)

	// Create gRPC server and run it in a separate goroutine.
	//grpcServer := grpcEngine.NewGRPCServer()
	//go grpcServer.Start()

	// Create HTTP server and run it in a separate goroutine.
	httpServer := httpEngine.NewServer(log) // httpEngine.NewServer가 새 설정을 사용하도록 수정 필요할 수 있음
	go func() {
		if err := httpServer.Start(); err != nil {
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
	//log.Info("gRPC server shutdown gracefully") // 이전 configs.Logger 대신 log 사용

	log.Info("Server exited")
}
