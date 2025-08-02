package http

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/middleware/auth"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

type PaymentHandler struct {
	usecase *usecase.PaymentUsecase
	logger  *zap.Logger
}

func NewPaymentHandler(usecase *usecase.PaymentUsecase, logger *zap.Logger) *PaymentHandler {
	return &PaymentHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *PaymentHandler) GetPayment(c echo.Context) error {
	id := c.Param("id")

	payment, err := h.usecase.GetPayment(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Payment not found",
		})
	}

	return c.JSON(http.StatusOK, payment)
}

func (h *PaymentHandler) GetUserPayments(c echo.Context) error {
	// Get authenticated user from JWT
	user, err := auth.RequireAuth(c)
	if err != nil {
		return err // RequireAuth already returns the JSON error response
	}

	h.logger.Info("Getting user payments",
		zap.String("user_id", user.UserID),
		zap.String("email", user.Email),
	)

	payments, err := h.usecase.GetUserPayments(c.Request().Context(), user.UserID)
	if err != nil {
		h.logger.Error("Failed to get user payments",
			zap.String("user_id", user.UserID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get payments",
		})
	}

	h.logger.Debug("Retrieved user payments",
		zap.String("user_id", user.UserID),
		zap.Int("payment_count", len(payments)),
	)

	return c.JSON(http.StatusOK, payments)
}

func (h *PaymentHandler) GetUserRecentPayments(c echo.Context) error {
	// Get authenticated user from JWT
	user, err := auth.RequireAuth(c)
	if err != nil {
		return err // RequireAuth already returns the JSON error response
	}

	// Parse limit query parameter
	limitStr := c.QueryParam("limit")
	limit := 10 // Default value
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			h.logger.Warn("Invalid limit parameter",
				zap.String("limit", limitStr),
				zap.Error(err))
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid limit parameter",
			})
		}
		limit = parsedLimit
	}

	h.logger.Info("Getting recent user payments",
		zap.String("user_id", user.UserID),
		zap.String("email", user.Email),
		zap.Int("limit", limit),
	)

	payments, err := h.usecase.GetUserRecentPayments(c.Request().Context(), user.UserID, limit)
	if err != nil {
		h.logger.Error("Failed to get recent user payments",
			zap.String("user_id", user.UserID),
			zap.Int("limit", limit),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get recent payments",
		})
	}

	h.logger.Debug("Retrieved recent user payments",
		zap.String("user_id", user.UserID),
		zap.Int("payment_count", len(payments)),
		zap.Int("requested_limit", limit),
	)

	return c.JSON(http.StatusOK, payments)
}
