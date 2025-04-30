package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/adapter/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/grpc"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/http"
	appinit "github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/init"
	"go.uber.org/zap"
)

func main() {
	// 1. 설정 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("설정 로드 실패: %v", err)
	}

	// 2. 로거 가져오기
	logger := cfg.Logger
	defer logger.Sync()

	logger.Info("인증 서비스를 시작합니다...",
		zap.String("service", cfg.Service.Name),
		zap.String("version", cfg.Service.Version),
	)

	// 3. 인프라스트럭처 초기화
	infrastructure, err := db.NewInfrastructure(cfg)
	if err != nil {
		logger.Fatal("인프라스트럭처 초기화 실패", zap.Error(err))
	}
	defer infrastructure.Close()

	// 4. 레포지토리 초기화
	repositories := repository.NewRepositories(infrastructure)

	// 5. 유스케이스 초기화
	useCases := appinit.NewUseCases(repositories, logger)

	// 나중에 사용하기 위해 전역 변수에 등록 또는 핸들러에 직접 주입
	// TODO: 추후 HTTP/gRPC 핸들러에 useCases 객체를 전달하는 로직 추가

	// 5. HTTP 서버 설정
	httpConfig := http.Config{
		Port:    cfg.Server.HTTP.Port,
		Timeout: cfg.Server.HTTP.Timeout,
		Debug:   cfg.Server.HTTP.Debug,
	}

	// 6. HTTP 서버 생성
	httpServer := http.NewServer(httpConfig, logger)
	httpServer.RegisterRoutes()

	// 7. gRPC 서버 설정
	grpcConfig := grpc.Config{
		Port:    cfg.Server.GRPC.Port,
		Timeout: cfg.Server.GRPC.Timeout,
	}

	// 8. gRPC 서버 생성
	grpcServer := grpc.NewServer(grpcConfig, logger)
	grpcServer.RegisterServices()

	// 9. 서버 시작
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

	// 10. 그레이스풀 종료를 위한 시그널 처리
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("서버를 종료합니다...")

	// 서버 종료
	if err := httpServer.Stop(); err != nil {
		logger.Error("HTTP 서버 종료 오류", zap.Error(err))
	}

	grpcServer.Stop()

	logger.Info("서버가 정상적으로 종료되었습니다")
}
