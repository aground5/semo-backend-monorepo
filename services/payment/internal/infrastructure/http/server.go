package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	"go.uber.org/zap"
)

type Server struct {
	config *config.Config
	logger *zap.Logger
	echo   *echo.Echo
}

func NewServer(cfg *config.Config, logger *zap.Logger) *Server {
	e := echo.New()
	
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	return &Server{
		config: cfg,
		logger: logger,
		echo:   e,
	}
}

func (s *Server) Start() error {
	// Setup routes
	s.setupRoutes()
	
	addr := fmt.Sprintf("%s:%d", s.config.Server.HTTP.Host, s.config.Server.HTTP.Port)
	s.logger.Info("Starting HTTP server", zap.String("address", addr))
	
	return s.echo.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

func (s *Server) setupRoutes() {
	// Health check
	s.echo.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
			"service": "payment",
		})
	})

	// API routes
	api := s.echo.Group("/api/v1")
	
	// Payment routes would be added here
	// api.POST("/payments", handler.CreatePayment)
	// api.GET("/payments/:id", handler.GetPayment)
	// etc.
	_ = api
}