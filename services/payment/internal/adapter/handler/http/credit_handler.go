package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/dto"
	customErr "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

// CreditHandler handles credit-related HTTP requests
type CreditHandler struct {
	logger                   *zap.Logger
	creditService            *usecase.CreditService
	creditTransactionService *usecase.CreditTransactionService
}

// NewCreditHandler creates a new credit handler instance
func NewCreditHandler(
	logger *zap.Logger,
	creditService *usecase.CreditService,
	creditTransactionService *usecase.CreditTransactionService,
) *CreditHandler {
	return &CreditHandler{
		logger:                   logger,
		creditService:            creditService,
		creditTransactionService: creditTransactionService,
	}
}

// GetUserCredits handles GET /api/v1/credits
func (h *CreditHandler) GetUserCredits(c echo.Context) error {
	// Extract user ID from JWT claims
	universalIDStr, ok := c.Get("universal_id").(string)
	if !ok {
		h.logger.Error("Failed to extract user ID from JWT claims")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	universalID, err := uuid.Parse(universalIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID format", zap.String("universal_id", universalIDStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID format",
		})
	}

	// Determine service provider, falling back to default if query param is empty
	serviceProvider := c.QueryParam("provider")

	// Get user's credit balance
	balance, err := h.creditService.GetBalanceForProvider(c.Request().Context(), universalID, serviceProvider)
	if err != nil {
		h.logger.Error("Failed to get user credit balance",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to retrieve credit balance",
		})
	}

	// Format response - return only current_balance
	response := map[string]interface{}{
		"current_balance": balance.CurrentBalance.String(),
	}

	return c.JSON(http.StatusOK, response)
}

// GetTransactionHistory handles GET /api/v1/credits/transactions
func (h *CreditHandler) GetTransactionHistory(c echo.Context) error {
	// Extract user ID from JWT claims
	universalIDStr, ok := c.Get("universal_id").(string)
	if !ok {
		h.logger.Error("Failed to extract user ID from JWT claims")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	universalID, err := uuid.Parse(universalIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID format", zap.String("universal_id", universalIDStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID format",
		})
	}

	// Parse query parameters
	filters := dto.TransactionFilters{
		UserID: universalID,
	}

	// Parse limit
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid limit parameter",
			})
		}
		filters.Limit = limit
	}

	// Parse offset
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid offset parameter",
			})
		}
		filters.Offset = offset
	}

	// Parse start date
	if startDateStr := c.QueryParam("start_date"); startDateStr != "" {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid start_date format, use ISO 8601",
			})
		}
		filters.StartDate = &startDate
	}

	// Parse end date
	if endDateStr := c.QueryParam("end_date"); endDateStr != "" {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid end_date format, use ISO 8601",
			})
		}
		filters.EndDate = &endDate
	}

	// Parse transaction type
	if transactionType := c.QueryParam("transaction_type"); transactionType != "" {
		// Validate transaction type
		validTypes := map[string]bool{
			"credit_allocation": true,
			"credit_usage":      true,
			"refund":            true,
			"adjustment":        true,
		}
		if !validTypes[transactionType] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid transaction_type, must be one of: credit_allocation, credit_usage, refund, adjustment",
			})
		}
		filters.TransactionType = &transactionType
	}

	// Get transaction history
	response, err := h.creditTransactionService.GetUserTransactionHistory(c.Request().Context(), universalID, filters)
	if err != nil {
		h.logger.Error("Failed to get transaction history",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to retrieve transaction history",
		})
	}

	return c.JSON(http.StatusOK, response)
}

// UseCredits handles POST /api/v1/credits
func (h *CreditHandler) UseCredits(c echo.Context) error {
	// Extract user ID from JWT claims
	universalIDStr, ok := c.Get("universal_id").(string)
	if !ok {
		h.logger.Error("Failed to extract user ID from JWT claims")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	universalID, err := uuid.Parse(universalIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID format", zap.String("universal_id", universalIDStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID format",
		})
	}

	// Parse request body
	var req dto.UseCreditRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	// Validate request
	if err := c.Validate(req); err != nil {
		h.logger.Error("Request validation failed", zap.Error(err))
		// Format validation errors for better user experience
		validationErr := "validation failed"
		if ve, ok := err.(validator.ValidationErrors); ok {
			validationErr = "validation failed: "
			for i, fe := range ve {
				if i > 0 {
					validationErr += ", "
				}
				switch fe.Tag() {
				case "required":
					validationErr += fmt.Sprintf("%s is required", fe.Field())
				case "min":
					validationErr += fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
				case "max":
					validationErr += fmt.Sprintf("%s must be at most %s characters", fe.Field(), fe.Param())
				case "uuid4":
					validationErr += fmt.Sprintf("%s must be a valid UUID", fe.Field())
				default:
					validationErr += fmt.Sprintf("%s failed %s validation", fe.Field(), fe.Tag())
				}
			}
		} else {
			validationErr = "validation failed: " + err.Error()
		}
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": validationErr,
		})
	}

	// Parse amount to decimal
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		h.logger.Error("Invalid amount format", zap.String("amount", req.Amount), zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid amount format",
		})
	}

	// Validate amount is positive
	if amount.LessThanOrEqual(decimal.Zero) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "amount must be greater than zero",
		})
	}

	// Parse idempotency key if provided
	var idempotencyKey *uuid.UUID
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		key, err := uuid.Parse(*req.IdempotencyKey)
		if err != nil {
			h.logger.Error("Invalid idempotency key format", zap.String("key", *req.IdempotencyKey), zap.Error(err))
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid idempotency key format",
			})
		}
		idempotencyKey = &key
	}

	// Convert usage metadata to JSON if provided
	var usageMetadata json.RawMessage
	if req.UsageMetadata != nil && len(req.UsageMetadata) > 0 {
		metadataBytes, err := json.Marshal(req.UsageMetadata)
		if err != nil {
			h.logger.Error("Failed to marshal usage metadata", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to process usage metadata",
			})
		}
		usageMetadata = metadataBytes
	}

	// Call service to use credits
	transaction, err := h.creditService.UseCredits(
		c.Request().Context(),
		universalID,
		req.ServiceProvider,
		amount,
		req.FeatureName,
		req.Description,
		usageMetadata,
		idempotencyKey,
	)

	// Handle specific errors
	if err != nil {
		// Check for insufficient balance error
		var insufficientErr *customErr.InsufficientBalanceError
		if errors.As(err, &insufficientErr) {
			h.logger.Warn("Insufficient credit balance",
				zap.String("universal_id", universalID.String()),
				zap.String("requested", amount.String()),
				zap.String("available", insufficientErr.Available.String()))
			return c.JSON(http.StatusPaymentRequired, map[string]string{
				"error":             "insufficient_credits",
				"message":           "Insufficient credit balance",
				"requested_amount":  amount.String(),
				"available_balance": insufficientErr.Available.String(),
			})
		}

		// Check for duplicate idempotency key
		if errors.Is(err, sql.ErrNoRows) {
			// This shouldn't happen for UseCredits, but handle it
			h.logger.Error("Unexpected no rows error", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to process credit usage",
			})
		}

		// Check if it's a duplicate transaction error (idempotency conflict)
		if idempotencyKey != nil {
			// Log as warning since it might be a retry
			h.logger.Warn("Possible duplicate credit usage attempt",
				zap.String("universal_id", universalID.String()),
				zap.String("idempotency_key", idempotencyKey.String()),
				zap.Error(err))
			// Could be a unique constraint violation
			return c.JSON(http.StatusConflict, map[string]string{
				"error":   "duplicate_request",
				"message": "A request with this idempotency key has already been processed",
			})
		}

		// Generic error
		h.logger.Error("Failed to use credits",
			zap.String("universal_id", universalID.String()),
			zap.String("amount", amount.String()),
			zap.String("feature", req.FeatureName),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to process credit usage",
		})
	}

	// Prepare successful response
	response := dto.UseCreditResponse{
		Success:       true,
		TransactionID: transaction.ID,
		BalanceAfter:  transaction.BalanceAfter.String(),
		Message:       "Credits successfully deducted",
	}

	h.logger.Info("Credits used successfully",
		zap.String("universal_id", universalID.String()),
		zap.Int64("transaction_id", transaction.ID),
		zap.String("amount", amount.String()),
		zap.String("feature", req.FeatureName),
		zap.String("balance_after", transaction.BalanceAfter.String()))

	return c.JSON(http.StatusOK, response)
}
