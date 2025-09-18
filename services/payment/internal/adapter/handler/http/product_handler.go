package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	providerFactory "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/middleware/auth"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

// ProductHandler handles one-time payment endpoints
type ProductHandler struct {
	productUseCase  *usecase.ProductUseCase
	providerFactory *providerFactory.Factory
	logger          *zap.Logger
}

// NewProductHandler creates a new ProductHandler instance
func NewProductHandler(
	productUseCase *usecase.ProductUseCase,
	providerFactory *providerFactory.Factory,
	logger *zap.Logger,
) *ProductHandler {
	return &ProductHandler{
		productUseCase:  productUseCase,
		providerFactory: providerFactory,
		logger:          logger,
	}
}

// CreatePaymentRequest represents the HTTP request for creating a payment
type CreatePaymentRequest struct {
	Amount      int64                  `json:"amount" validate:"required,min=100"`
	Currency    string                 `json:"currency" validate:"required"`
	OrderName   string                 `json:"order_name" validate:"required"`
	CustomerKey string                 `json:"customer_key"`
	PlanID      string                 `json:"plan_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreatePayment handles POST /products endpoint
func (h *ProductHandler) CreatePayment(c echo.Context) error {
	ctx := c.Request().Context()

	// Get provider from query parameter (default: toss)
	providerStr := c.QueryParam("provider")
	if providerStr == "" {
		providerStr = string(provider.ProviderTypeToss)
	}

	// Get provider instance
	paymentProvider, err := h.providerFactory.GetProviderFromString(providerStr)
	if err != nil {
		h.logger.Error("Failed to get payment provider",
			zap.String("provider", providerStr),
			zap.Error(err))

		if providerStr == string(provider.ProviderTypeStripe) {
			return c.JSON(http.StatusNotImplemented, echo.Map{
				"error": "Stripe one-time payment is not yet implemented",
				"code":  "PROVIDER_NOT_IMPLEMENTED",
			})
		}

		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid payment provider",
			"code":  "INVALID_PROVIDER",
		})
	}

	// Parse request body
	var req CreatePaymentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request",
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate request
	if err := c.Validate(req); err != nil {
		h.logger.Error("Failed to validate request",
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Validation failed",
			"code":  "VALIDATION_FAILED",
			"details": err.Error(),
		})
	}

	// Get universal ID from JWT context
	universalID, err := auth.GetUniversalID(c)
	if err != nil {
		h.logger.Error("Failed to get universal ID",
			zap.Error(err))
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error": "Failed to get user information",
			"code":  "AUTH_ERROR",
		})
	}

	// Create payment request
	usecaseReq := &usecase.CreatePaymentRequest{
		UniversalID: universalID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		OrderName:   req.OrderName,
		CustomerKey: req.CustomerKey,
		PlanID:      req.PlanID,
		Metadata:    req.Metadata,
	}

	// Create payment with provider
	resp, err := h.productUseCase.CreatePaymentWithProvider(ctx, usecaseReq, paymentProvider)
	if err != nil {
		h.logger.Error("Failed to create payment",
			zap.String("universal_id", universalID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to create payment",
			"code":  "PAYMENT_CREATION_FAILED",
		})
	}

	h.logger.Info("Payment created successfully",
		zap.String("order_id", resp.OrderID),
		zap.String("universal_id", universalID),
		zap.String("provider", providerStr))

	return c.JSON(http.StatusCreated, resp)
}

// ConfirmPaymentRequest represents the HTTP request for confirming a payment
type ConfirmPaymentRequest struct {
	PaymentKey string                 `json:"paymentKey" validate:"required"`
	OrderID    string                 `json:"orderId" validate:"required"`
	Amount     int64                  `json:"amount" validate:"required"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ConfirmPayment handles POST /products/confirm endpoint
func (h *ProductHandler) ConfirmPayment(c echo.Context) error {
	ctx := c.Request().Context()

	// Get provider from query parameter (default: toss)
	providerStr := c.QueryParam("provider")
	if providerStr == "" {
		providerStr = string(provider.ProviderTypeToss)
	}

	// Get provider instance
	paymentProvider, err := h.providerFactory.GetProviderFromString(providerStr)
	if err != nil {
		h.logger.Error("Failed to get payment provider",
			zap.String("provider", providerStr),
			zap.Error(err))

		if providerStr == string(provider.ProviderTypeStripe) {
			return c.JSON(http.StatusNotImplemented, echo.Map{
				"error": "Stripe payment confirmation is not yet implemented",
				"code":  "PROVIDER_NOT_IMPLEMENTED",
			})
		}

		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid payment provider",
			"code":  "INVALID_PROVIDER",
		})
	}

	// Parse request body
	var req ConfirmPaymentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request",
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate request
	if err := c.Validate(req); err != nil {
		h.logger.Error("Failed to validate request",
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Validation failed",
			"code":  "VALIDATION_FAILED",
			"details": err.Error(),
		})
	}

	// Get universal ID from JWT context (for logging)
	universalID, _ := auth.GetUniversalID(c)

	// Create confirmation request
	usecaseReq := &usecase.ConfirmPaymentRequest{
		OrderID:    req.OrderID,
		PaymentKey: req.PaymentKey,
		Amount:     req.Amount,
		Provider:   providerStr,
		Metadata:   req.Metadata,
	}

	// Confirm payment with provider
	resp, err := h.productUseCase.ConfirmPaymentWithProvider(ctx, usecaseReq, paymentProvider)
	if err != nil {
		h.logger.Error("Failed to confirm payment",
			zap.String("order_id", req.OrderID),
			zap.String("universal_id", universalID),
			zap.Error(err))

		// Check if it's a provider error
		if providerErr, ok := err.(*provider.ProviderError); ok {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": providerErr.Message,
				"code":  providerErr.Code,
			})
		}

		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to confirm payment",
			"code":  "PAYMENT_CONFIRMATION_FAILED",
		})
	}

	h.logger.Info("Payment confirmed successfully",
		zap.String("order_id", resp.OrderID),
		zap.String("transaction_key", resp.TransactionKey),
		zap.String("universal_id", universalID),
		zap.String("provider", providerStr))

	return c.JSON(http.StatusOK, resp)
}

