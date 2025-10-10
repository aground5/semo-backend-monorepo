package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stripe/stripe-go/v79"
	handlers "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/handler/http"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/database"
	providerFactory "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/middleware/auth"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

// CustomValidator implements echo.Validator interface
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates the input struct
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return err
	}
	return nil
}

type Server struct {
	config *config.Config
	logger *zap.Logger
	echo   *echo.Echo
	repos  *database.Repositories
}

func NewServer(cfg *config.Config, logger *zap.Logger, repos *database.Repositories) *Server {
	e := echo.New()

	// Register custom validator
	e.Validator = &CustomValidator{validator: validator.New()}

	// Initialize Stripe
	stripe.Key = cfg.Service.StripeSecretKey

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.Service.AllowedClientOrigins(),
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

	// Initialize provider factory
	factory := providerFactory.NewFactory(s.config, s.logger)

	// Initialize services
	subscriptionService := usecase.NewSubscriptionService(s.repos.CustomerMapping, s.repos.Subscription, s.logger)
	creditService := usecase.NewCreditService(s.repos.Credit, s.repos.Subscription, s.repos.Plan, s.logger, model.ServiceProviderSemo)
	creditTransactionService := usecase.NewCreditTransactionService(s.repos.CreditTransaction, s.logger, model.ServiceProviderSemo)
	workspaceVerificationService := usecase.NewWorkspaceVerificationService(s.repos.WorkspaceVerification, s.logger)
	productUseCase := usecase.NewProductUseCase(s.repos.Payment, s.logger)

	// Initialize handlers
	plansHandler := handlers.NewPlansHandler(s.logger, s.repos.Plan)
	checkoutHandler := handlers.NewCheckoutHandler(s.logger, s.config.Service.PrimaryClientURL(), s.config.Service.AllowedClientOrigins(), s.repos.CustomerMapping)
	subscriptionHandler := handlers.NewSubscriptionHandler(s.logger, subscriptionService, s.repos.CustomerMapping, s.config.Service.PrimaryClientURL())
	webhookHandler := handlers.NewWebhookHandler(s.logger, s.config.Service.StripeWebhookSecret, s.repos.Webhook, s.repos.Subscription, s.repos.Payment, s.repos.CustomerMapping, s.repos.Credit, s.repos.Plan, model.ServiceProviderSemo)
	paymentUsecase := usecase.NewPaymentUsecase(s.repos.Payment, nil, s.logger)
	paymentHandler := handlers.NewPaymentHandler(paymentUsecase, s.logger)
	creditHandler := handlers.NewCreditHandler(s.logger, creditService, creditTransactionService)
	productHandler := handlers.NewProductHandler(productUseCase, factory, s.repos.CustomerMapping, s.logger)
	tossWebhookHandler := handlers.NewTossWebhookHandler(s.logger, s.repos.Payment, s.config.Service.Toss.SecretKey, s.config.Service.Toss.ClientKey)

	// JWT middleware configuration
	jwtConfig := auth.JWTConfig{
		Secret:                       s.config.Service.Supabase.JWTSecret,
		Logger:                       s.logger,
		WorkspaceVerificationService: workspaceVerificationService,
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
	v1.GET("/plans/subscription", plansHandler.GetSubscriptionPlans) // Subscription-type payment plans only
	v1.GET("/plans/one-time", plansHandler.GetOneTimePlans)          // One-time payment plans only

	// Protected routes (require JWT authentication)
	protected := v1.Group("", auth.JWTMiddleware(jwtConfig))

	// Subscriptions - RESTful style (all require authentication)
	subscriptions := protected.Group("/subscriptions")
	subscriptions.POST("", subscriptionHandler.CreateSubscription)
	subscriptions.GET("/current", subscriptionHandler.GetCurrentSubscription)
	subscriptions.DELETE("/current", subscriptionHandler.CancelCurrentSubscription) // New secure endpoint
	subscriptions.POST("/portal", checkoutHandler.CreatePortalSession)

	// One-time payment - RESTful style (all require authentication)
	products := protected.Group("/products")
	products.POST("", productHandler.CreateProduct)          // Provider-based payment creation
	products.POST("/confirm", productHandler.ConfirmProduct) // Provider payment confirmation

	// Checkout session status endpoint (requires authentication)
	protected.GET("/checkout/session/:sessionId", checkoutHandler.CheckSessionStatus)

	// Payment routes (require authentication)
	protected.GET("/payments", paymentHandler.GetPayments)
	protected.GET("/payments/:id", paymentHandler.GetPaymentByTxID)

	// Credit routes (require authentication)
	protected.GET("/credits", creditHandler.GetUserCredits)
	protected.POST("/credits", creditHandler.UseCredits)
	protected.GET("/credits/transactions", creditHandler.GetTransactionHistory)

	// Internal/Debug routes
	internal := v1.Group("/internal")
	internal.GET("/webhook-data", webhookHandler.GetWebhookData)

	// Webhook routes (outside API versioning)
	s.echo.POST("/webhook", webhookHandler.HandleWebhook)   // Stripe webhook
	s.echo.POST("/webhook/toss", tossWebhookHandler.Handle) // Toss webhook
}
