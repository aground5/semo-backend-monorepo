package http

import (
	"net/http"

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
