package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

type BillingHandler struct {
	billingService *usecase.BillingService
	logger         *zap.Logger
}

func NewBillingHandler(billingService *usecase.BillingService, logger *zap.Logger) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
		logger:         logger,
	}
}

type issueBillingKeyRequest struct {
	AuthKey     string `json:"auth_key" validate:"required"`
	CustomerKey string `json:"customer_key" validate:"required"`
}

type billingKeyResponse struct {
	ID           int64     `json:"id"`
	CardLastFour string    `json:"card_last_four"`
	CardCompany  string    `json:"card_company"`
	CardType     string    `json:"card_type"`
	CreatedAt    time.Time `json:"created_at"`
}

// IssueBillingKey handles POST /api/v1/billing/issue
func (h *BillingHandler) IssueBillingKey(c echo.Context) error {
	universalIDStr, ok := c.Get("universal_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
	}

	universalID, err := uuid.Parse(universalIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user ID"})
	}

	var req issueBillingKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "auth_key and customer_key are required"})
	}

	billingKey, err := h.billingService.IssueBillingKey(
		c.Request().Context(),
		universalID,
		req.AuthKey,
		req.CustomerKey,
		c.RealIP(),
		c.Request().UserAgent(),
	)
	if err != nil {
		h.logger.Error("failed to issue billing key",
			zap.String("universal_id", universalIDStr),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, billingKeyResponse{
		ID:           billingKey.ID,
		CardLastFour: billingKey.CardLastFour,
		CardCompany:  billingKey.CardCompany,
		CardType:     billingKey.CardType,
		CreatedAt:    billingKey.CreatedAt,
	})
}

// GetCards handles GET /api/v1/billing/cards
func (h *BillingHandler) GetCards(c echo.Context) error {
	universalIDStr, ok := c.Get("universal_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
	}

	universalID, err := uuid.Parse(universalIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user ID"})
	}

	cards, err := h.billingService.GetCards(c.Request().Context(), universalID)
	if err != nil {
		h.logger.Error("failed to get cards",
			zap.String("universal_id", universalIDStr),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get cards"})
	}

	result := make([]billingKeyResponse, len(cards))
	for i, card := range cards {
		result[i] = billingKeyResponse{
			ID:           card.ID,
			CardLastFour: card.CardLastFour,
			CardCompany:  card.CardCompany,
			CardType:     card.CardType,
			CreatedAt:    card.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, echo.Map{"cards": result})
}

// DeactivateCard handles DELETE /api/v1/billing/cards/:id
func (h *BillingHandler) DeactivateCard(c echo.Context) error {
	universalIDStr, ok := c.Get("universal_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
	}

	universalID, err := uuid.Parse(universalIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user ID"})
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid card ID"})
	}

	err = h.billingService.DeactivateCard(
		c.Request().Context(),
		id,
		universalID,
		c.RealIP(),
		c.Request().UserAgent(),
	)
	if err != nil {
		h.logger.Error("failed to deactivate card",
			zap.Int64("card_id", id),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"status": "deactivated"})
}

type chargeBillingKeyRequest struct {
	BillingKeyID    int64  `json:"billing_key_id" validate:"required"`
	Amount          int64  `json:"amount" validate:"required,gt=0"`
	OrderName       string `json:"order_name" validate:"required"`
	PlanID          string `json:"plan_id" validate:"required"`
	ServiceProvider string `json:"service_provider"`
}

type chargeBillingKeyResponse struct {
	OrderID          string `json:"order_id"`
	PaymentKey       string `json:"payment_key"`
	TransactionKey   string `json:"transaction_key"`
	Status           string `json:"status"`
	Amount           int64  `json:"amount"`
	ApprovedAt       string `json:"approved_at,omitempty"`
	CreditsAllocated int    `json:"credits_allocated"`
}

// ChargeBillingKey handles POST /api/v1/billing/charge
func (h *BillingHandler) ChargeBillingKey(c echo.Context) error {
	universalIDStr, ok := c.Get("universal_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
	}

	universalID, err := uuid.Parse(universalIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user ID"})
	}

	var req chargeBillingKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "validation failed",
			"details": err.Error(),
		})
	}

	result, err := h.billingService.ChargeBillingKey(
		c.Request().Context(),
		universalID,
		req.BillingKeyID,
		req.Amount,
		req.OrderName,
		req.PlanID,
		req.ServiceProvider,
		c.RealIP(),
		c.Request().UserAgent(),
	)
	if err != nil {
		h.logger.Error("failed to charge billing key",
			zap.Int64("billing_key_id", req.BillingKeyID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	resp := chargeBillingKeyResponse{
		OrderID:          result.OrderID,
		PaymentKey:       result.PaymentKey,
		TransactionKey:   result.TransactionKey,
		Status:           result.Status,
		Amount:           result.Amount,
		CreditsAllocated: result.CreditsAllocated,
	}
	if result.ApprovedAt != nil {
		resp.ApprovedAt = result.ApprovedAt.Format(time.RFC3339)
	}

	return c.JSON(http.StatusOK, resp)
}
