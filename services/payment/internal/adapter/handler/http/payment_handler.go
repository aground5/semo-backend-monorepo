package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
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

func (h *PaymentHandler) CreatePayment(c echo.Context) error {
	// Implementation
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Payment created successfully",
	})
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
	userID := c.QueryParam("user_id")
	
	payments, err := h.usecase.GetUserPayments(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get payments",
		})
	}
	
	return c.JSON(http.StatusOK, payments)
}