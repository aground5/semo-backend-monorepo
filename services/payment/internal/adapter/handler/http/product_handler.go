package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	providerFactory "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/middleware/auth"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

// ProductHandler handles one-time payment endpoints
type ProductHandler struct {
	productUseCase      *usecase.ProductUseCase
	providerFactory     *providerFactory.Factory
	customerMappingRepo domainRepo.CustomerMappingRepository
	logger              *zap.Logger
}

// NewProductHandler creates a new ProductHandler instance
func NewProductHandler(
	productUseCase *usecase.ProductUseCase,
	providerFactory *providerFactory.Factory,
	customerMappingRepo domainRepo.CustomerMappingRepository,
	logger *zap.Logger,
) *ProductHandler {
	return &ProductHandler{
		productUseCase:      productUseCase,
		providerFactory:     providerFactory,
		customerMappingRepo: customerMappingRepo,
		logger:              logger,
	}
}

// ensureProviderCustomerMapping creates or updates the mapping between a universal user ID and a provider customer ID.
func (h *ProductHandler) ensureProviderCustomerMapping(ctx context.Context, providerName string, universalID string, providerCustomerID string, email string) {
	if h.customerMappingRepo == nil {
		return
	}

	if providerCustomerID == "" {
		h.logger.Warn("Skipping customer mapping creation: missing provider customer ID",
			zap.String("provider", providerName),
			zap.String("universal_id", universalID))
		return
	}

	existingByUser, err := h.customerMappingRepo.GetByProviderAndUniversalID(ctx, providerName, universalID)
	if err != nil {
		h.logger.Warn("Failed to check existing customer mapping by user",
			zap.String("provider", providerName),
			zap.String("universal_id", universalID),
			zap.Error(err))
		return
	}

	if existingByUser != nil {
		needsUpdate := existingByUser.ProviderCustomerID != providerCustomerID
		if email != "" && existingByUser.Email == "" {
			existingByUser.Email = email
			needsUpdate = true
		}
		if needsUpdate {
			existingByUser.ProviderCustomerID = providerCustomerID
			if err := h.customerMappingRepo.Update(ctx, existingByUser); err != nil {
				h.logger.Warn("Failed to update customer mapping for provider",
					zap.String("provider", providerName),
					zap.String("universal_id", universalID),
					zap.String("provider_customer_id", providerCustomerID),
					zap.Error(err))
			} else {
				h.logger.Info("Updated customer mapping for provider",
					zap.String("provider", providerName),
					zap.String("universal_id", universalID),
					zap.String("provider_customer_id", providerCustomerID))
			}
		}
		return
	}

	existingByCustomer, err := h.customerMappingRepo.GetByProviderCustomerID(ctx, providerName, providerCustomerID)
	if err != nil {
		h.logger.Warn("Failed to check existing customer mapping by provider ID",
			zap.String("provider", providerName),
			zap.String("provider_customer_id", providerCustomerID),
			zap.Error(err))
		return
	}

	if existingByCustomer != nil {
		if existingByCustomer.UniversalID != universalID {
			h.logger.Warn("Provider customer ID already mapped to different user",
				zap.String("provider", providerName),
				zap.String("provider_customer_id", providerCustomerID),
				zap.String("existing_universal_id", existingByCustomer.UniversalID),
				zap.String("requested_universal_id", universalID))
		}
		return
	}

	mapping := &entity.CustomerMapping{
		Provider:           providerName,
		ProviderCustomerID: providerCustomerID,
		UniversalID:        universalID,
		Email:              email,
	}

	if err := h.customerMappingRepo.Create(ctx, mapping); err != nil {
		h.logger.Warn("Failed to create customer mapping for provider",
			zap.String("provider", providerName),
			zap.String("provider_customer_id", providerCustomerID),
			zap.String("universal_id", universalID),
			zap.Error(err))
		return
	}

	h.logger.Info("Customer mapping created for provider",
		zap.String("provider", providerName),
		zap.String("provider_customer_id", providerCustomerID),
		zap.String("universal_id", universalID))
}

// CreateProductRequest represents the HTTP request for creating a payment
type CreateProductRequest struct {
	Amount      int64                  `json:"amount" validate:"required,min=100"`
	Currency    string                 `json:"currency" validate:"required"`
	OrderName   string                 `json:"order_name" validate:"required"`
	CustomerKey string                 `json:"customer_key"`
	PlanID      string                 `json:"plan_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateProduct handles POST /products endpoint
func (h *ProductHandler) CreateProduct(c echo.Context) error {
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
	var req CreateProductRequest
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
			"error":   "Validation failed",
			"code":    "VALIDATION_FAILED",
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

	var metadataEmail string
	if req.Metadata != nil {
		if emailValue, ok := req.Metadata["email"].(string); ok {
			metadataEmail = emailValue
		}
	}

	if providerStr == string(provider.ProviderTypeToss) {
		h.ensureProviderCustomerMapping(ctx, providerStr, universalID, req.CustomerKey, metadataEmail)
	}

	// Create payment request
	usecaseReq := &usecase.CreateProductRequest{
		UniversalID: universalID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		OrderName:   req.OrderName,
		CustomerKey: req.CustomerKey,
		PlanID:      req.PlanID,
		Metadata:    req.Metadata,
	}

	// Create payment with provider
	resp, err := h.productUseCase.CreateProductWithProvider(ctx, usecaseReq, paymentProvider)
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

// ConfirmProductRequest represents the HTTP request for confirming a payment
type ConfirmProductRequest struct {
	PaymentKey string                 `json:"paymentKey" validate:"required"`
	OrderID    string                 `json:"orderId" validate:"required"`
	Amount     int64                  `json:"amount" validate:"required"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ConfirmPayment handles POST /products/confirm endpoint
func (h *ProductHandler) ConfirmProduct(c echo.Context) error {
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
	var req ConfirmProductRequest
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
			"error":   "Validation failed",
			"code":    "VALIDATION_FAILED",
			"details": err.Error(),
		})
	}

	// Get universal ID from JWT context (for logging)
	universalID, _ := auth.GetUniversalID(c)

	// Create confirmation request
	usecaseReq := &usecase.ConfirmProductRequest{
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
