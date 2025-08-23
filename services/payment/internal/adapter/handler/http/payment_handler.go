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

func (h *PaymentHandler) GetUserPayments(c echo.Context) error {
	// Get authenticated user from JWT
	user, err := auth.RequireAuth(c)
	if err != nil {
		return err // RequireAuth already returns the JSON error response
	}

	// Parse pagination parameters
	page := 1
	limit := 20
	
	// Parse page parameter
	if pageStr := c.QueryParam("page"); pageStr != "" {
		parsedPage, err := strconv.Atoi(pageStr)
		if err != nil {
			h.logger.Warn("Invalid page parameter",
				zap.String("page", pageStr),
				zap.Error(err))
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid page parameter",
			})
		}
		if parsedPage < 1 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Page must be greater than 0",
			})
		}
		page = parsedPage
	}
	
	// Parse limit parameter
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			h.logger.Warn("Invalid limit parameter",
				zap.String("limit", limitStr),
				zap.Error(err))
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid limit parameter",
			})
		}
		if parsedLimit < 1 || parsedLimit > 100 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Limit must be between 1 and 100",
			})
		}
		limit = parsedLimit
	}

	h.logger.Info("Getting user payments",
		zap.String("user_id", user.UserID),
		zap.String("email", user.Email),
		zap.Int("page", page),
		zap.Int("limit", limit),
	)

	response, err := h.usecase.GetUserPayments(c.Request().Context(), user.UserID, page, limit)
	if err != nil {
		h.logger.Error("Failed to get user payments",
			zap.String("user_id", user.UserID),
			zap.Int("page", page),
			zap.Int("limit", limit),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get payments",
		})
	}

	h.logger.Debug("Retrieved user payments",
		zap.String("user_id", user.UserID),
		zap.Int("payment_count", len(response.Data)),
		zap.Int64("total_count", response.Pagination.Total),
		zap.Int("current_page", response.Pagination.CurrentPage),
		zap.Int("total_pages", response.Pagination.TotalPages),
	)

	return c.JSON(http.StatusOK, response)
}

func (h *PaymentHandler) GetPaymentByTxID(c echo.Context) error {
	id := c.Param("id")

	payment, err := h.usecase.GetPayment(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Payment not found",
		})
	}

	return c.JSON(http.StatusOK, payment)
}