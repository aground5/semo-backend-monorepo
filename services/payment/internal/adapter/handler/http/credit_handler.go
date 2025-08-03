package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

// CreditHandler handles credit-related HTTP requests
type CreditHandler struct {
	logger        *zap.Logger
	creditService *usecase.CreditService
}

// NewCreditHandler creates a new credit handler instance
func NewCreditHandler(logger *zap.Logger, creditService *usecase.CreditService) *CreditHandler {
	return &CreditHandler{
		logger:        logger,
		creditService: creditService,
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