package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/dto"
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
	userIDStr, ok := c.Get("user_id").(string)
	if !ok {
		h.logger.Error("Failed to extract user ID from JWT claims")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID format", zap.String("user_id", userIDStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID format",
		})
	}

	// Get user's credit balance
	balance, err := h.creditService.GetBalance(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user credit balance", 
			zap.String("user_id", userID.String()),
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
	userIDStr, ok := c.Get("user_id").(string)
	if !ok {
		h.logger.Error("Failed to extract user ID from JWT claims")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID format", zap.String("user_id", userIDStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID format",
		})
	}

	// Parse query parameters
	filters := dto.TransactionFilters{
		UserID: userID,
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
	response, err := h.creditTransactionService.GetUserTransactionHistory(c.Request().Context(), userID, filters)
	if err != nil {
		h.logger.Error("Failed to get transaction history",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to retrieve transaction history",
		})
	}

	return c.JSON(http.StatusOK, response)
}