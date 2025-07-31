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
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/middleware/auth"
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
	subscriptionHandler := handlers.NewSubscriptionHandler(s.logger, s.repos.CustomerMapping)
	webhookHandler := handlers.NewWebhookHandler(s.logger, s.config.Service.StripeWebhookSecret, s.repos.Webhook, s.repos.Subscription, s.repos.Payment, s.repos.CustomerMapping, s.repos.Plan)
	paymentUsecase := usecase.NewPaymentUsecase(s.repos.Payment, nil, s.logger)
	paymentHandler := handlers.NewPaymentHandler(paymentUsecase, s.logger)

	// JWT middleware configuration
	jwtConfig := auth.JWTConfig{
		Secret: s.config.Service.Supabase.JWTSecret,
		Logger: s.logger,
		SkipPaths: []string{
			"/health",
			"/webhook",
			"/api/v1/plans",
			"/api/v1/internal/webhook-data",
		},
	}

	// API v1 routes
	v1 := s.echo.Group("/api/v1")

	// Public routes (no authentication required)
	// Plans & Pricing - public for browsing
	v1.GET("/plans", plansHandler.GetPlans)                          // All plans (backward compatibility)
	v1.GET("/plans/subscription", plansHandler.GetSubscriptionPlans) // Subscription plans only
	v1.GET("/plans/one-time", plansHandler.GetOneTimePlans)          // One-time payment plans only

	// Protected routes (require JWT authentication)
	protected := v1.Group("", auth.JWTMiddleware(jwtConfig))

	// Subscriptions - RESTful style (all require authentication)
	subscriptions := protected.Group("/subscriptions")
	subscriptions.POST("", checkoutHandler.CreateSubscription)
	subscriptions.GET("/current", subscriptionHandler.GetCurrentSubscription)
	subscriptions.DELETE("/:id", subscriptionHandler.CancelSubscription)
	subscriptions.POST("/portal", checkoutHandler.CreatePortalSession)
	
	// Checkout session status endpoint (requires authentication)
	protected.GET("/checkout/session/:sessionId", checkoutHandler.CheckSessionStatus)

	// Payment routes (require authentication)
	protected.GET("/payments/:id", paymentHandler.GetPayment)
	protected.GET("/payments", paymentHandler.GetUserPayments)

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
