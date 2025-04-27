package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/your-org/semo-backend-monorepo/pkg/logger"
	"go.uber.org/zap"
)

// Server HTTP 서버 구조체
type Server struct {
	router  *echo.Echo
	server  *http.Server
	logger  *zap.Logger
	address string
}

// Config HTTP 서버 설정
type Config struct {
	Port    string
	Timeout int
	Debug   bool
}

// NewServer HTTP 서버 생성
func NewServer(cfg Config, zapLogger *zap.Logger) *Server {
	// Echo 인스턴스 생성
	e := echo.New()

	// 기본 미들웨어 설정
	e.Use(middleware.Recover())

	// 로그 미들웨어 설정
	e.Use(logger.NewEchoRequestLogger(zapLogger))

	// Echo 로거 설정
	logger.WithEchoLogger(e, zapLogger)

	// HTTP 서버 주소 설정
	address := fmt.Sprintf(":%s", cfg.Port)

	// HTTP 서버 설정
	server := &http.Server{
		Addr:         address,
		ReadTimeout:  time.Duration(cfg.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Timeout) * time.Second,
	}

	return &Server{
		router:  e,
		server:  server,
		logger:  zapLogger,
		address: address,
	}
}

// Router Echo 인스턴스 반환
func (s *Server) Router() *echo.Echo {
	return s.router
}

// RegisterRoutes HTTP 라우트 등록
func (s *Server) RegisterRoutes() {
	// 헬스 체크
	s.router.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// API 버전 그룹
	v1 := s.router.Group("/api/v1")

	// 예제 라우트
	v1.GET("/example", func(c echo.Context) error {
		return c.String(http.StatusOK, "API 버전 1 예제 엔드포인트")
	})

	// 인증 라우트
	v1.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	// TODO: API 라우트 등록
	// 예: auth.RegisterRoutes(v1, authHandler)
}

// Start HTTP 서버 시작
func (s *Server) Start() error {
	s.logger.Info("HTTP 서버 시작",
		zap.String("address", s.address),
	)

	// 서버 시작
	s.server.Handler = s.router
	return s.router.StartServer(s.server)
}

// Stop HTTP 서버 종료
func (s *Server) Stop() error {
	s.logger.Info("HTTP 서버 종료 중...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.router.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP 서버 종료 실패: %w", err)
	}

	s.logger.Info("HTTP 서버 종료 완료")
	return nil
}
