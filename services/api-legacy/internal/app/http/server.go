package httpEngine

import (
	"net/http"
	"semo-server/configs"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/wekeepgrowing/semo-backend-monorepo/pkg/logger" // pkg/logger 패키지 임포트 추가
	"go.uber.org/zap"

	"context"
)

type Server struct {
	e *echo.Echo
}

func NewServer(log *zap.Logger) *Server {
	e := echo.New()
	e.IPExtractor = echo.ExtractIPFromRealIPHeader()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://semo.world", "https://www.semo.world", "https://app.semo.world", "http://localhost:3000"}, // 특정 출처만 허용
		AllowCredentials: true,                                                                                                        // 쿠키 전송 허용
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderCacheControl, "x-duid", "x-anonymous-id"},
	}))

	e.Use(logger.NewEchoRequestLogger(log))
	
	logger.WithEchoLogger(e, log)

	e.Use(middleware.Recover())

	RegisterRoutes(e, log)

	return &Server{e: e}
}

// Start runs the Echo server on the configured HTTP port.
func (s *Server) Start() error {
	port := configs.Configs.Service.HttpPort
	if port == "" {
		port = "8080"
	}
	return s.e.Start(":" + port)
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.e.Shutdown(ctx)
}