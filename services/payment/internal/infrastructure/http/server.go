package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stripe/stripe-go/v76"
	handlers "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/handler/http"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/database"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

type Server struct {
	config *config.Config
	logger *zap.Logger
	echo   *echo.Echo
	repos  *database.Repositories
}

func NewServer(cfg *config.Config, logger *zap.Logger, repos *database.Repositories) *Server {
	e := echo.New()

	// Initialize Stripe
	stripe.Key = cfg.Service.StripeSecretKey

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{cfg.Service.ClientURL},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	return &Server{
		config: cfg,
		logger: logger,
		echo:   e,
		repos:  repos,
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
			"status":  "healthy",
			"service": "payment",
		})
	})

	// Initialize handlers
	plansHandler := handlers.NewPlansHandler(s.logger, s.repos.Plan)
	checkoutHandler := handlers.NewCheckoutHandler(s.logger, s.config.Service.ClientURL, s.repos.CustomerMapping)
	subscriptionHandler := handlers.NewSubscriptionHandler(s.logger)
	webhookHandler := handlers.NewWebhookHandler(s.logger, s.config.Service.StripeWebhookSecret, s.repos.Webhook, s.repos.Subscription, s.repos.Payment, s.repos.CustomerMapping, s.repos.Plan)
	paymentUsecase := usecase.NewPaymentUsecase(s.repos.Payment, nil, s.logger)
	paymentHandler := handlers.NewPaymentHandler(paymentUsecase, s.logger)

	// API v1 routes
	v1 := s.echo.Group("/api/v1")

	// Plans & Pricing
	v1.GET("/plans", plansHandler.GetPlans)                          // All plans (backward compatibility)
	v1.GET("/plans/subscription", plansHandler.GetSubscriptionPlans) // Subscription plans only
	v1.GET("/plans/one-time", plansHandler.GetOneTimePlans)          // One-time payment plans only

	// Subscriptions - RESTful style
	subscriptions := v1.Group("/subscriptions")
	subscriptions.POST("", checkoutHandler.CreateSubscription)
	subscriptions.GET("/current", subscriptionHandler.GetCurrentSubscription)
	subscriptions.DELETE("/:id", subscriptionHandler.CancelSubscription)
	subscriptions.POST("/portal", checkoutHandler.CreatePortalSession)

	// Payment routes (existing)
	v1.GET("/payments/:id", paymentHandler.GetPayment)
	v1.GET("/payments", paymentHandler.GetUserPayments)

	// Internal/Debug routes
	internal := v1.Group("/internal")
	internal.GET("/webhook-data", webhookHandler.GetWebhookData)

	// Audit test endpoint (for development/testing only)
	if s.config.Service.Environment != "production" {
		auditTestHandler := handlers.NewAuditTestHandler(s.logger, s.repos.Payment)
		internal.POST("/test-audit-log", auditTestHandler.TestAuditLog)
	}

	// Webhook route (outside API versioning)
	s.echo.POST("/webhook", webhookHandler.HandleWebhook)
}
