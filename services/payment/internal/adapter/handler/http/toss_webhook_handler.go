package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	tossProvider "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider/toss"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TossWebhookHandler handles TossPayments webhook events
type TossWebhookHandler struct {
	logger       *zap.Logger
	paymentRepo  repository.PaymentRepository
	tossProvider *tossProvider.TossProvider
}

// NewTossWebhookHandler creates a new TossWebhookHandler instance
func NewTossWebhookHandler(
	logger *zap.Logger,
	paymentRepo repository.PaymentRepository,
	secretKey string,
	clientKey string,
) *TossWebhookHandler {
	return &TossWebhookHandler{
		logger:       logger,
		paymentRepo:  paymentRepo,
		tossProvider: tossProvider.NewTossProvider(secretKey, clientKey, logger),
	}
}

// Handle processes TossPayments webhook events
func (h *TossWebhookHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	// Read request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body",
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Failed to read request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Get signature from header (if TossPayments provides one)
	signature := c.Request().Header.Get("X-Toss-Signature")

	var webhookPayload TossWebhookPayload
	if err := json.Unmarshal(body, &webhookPayload); err != nil {
		h.logger.Warn("Failed to parse Toss webhook payload for logging",
			zap.Error(err))
	} else {
		h.logger.Info("Processing Toss webhook event",
			zap.String("event_type", webhookPayload.EventType),
			zap.String("order_id", webhookPayload.Data.OrderID),
			zap.String("payment_key", webhookPayload.Data.PaymentKey),
			zap.String("status", webhookPayload.Data.Status))
		if webhookPayload.Data.Failure != nil {
			h.logger.Warn("Toss webhook failure details",
				zap.String("order_id", webhookPayload.Data.OrderID),
				zap.Any("failure", webhookPayload.Data.Failure))
		}
	}

	// Process webhook event with provider
	event, err := h.tossProvider.HandleWebhook(ctx, body, signature)
	if err != nil {
		h.logger.Error("Failed to process webhook",
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Failed to process webhook",
			"code":  "WEBHOOK_PROCESSING_FAILED",
		})
	}

	h.logger.Debug("Toss webhook event payload parsed",
		zap.String("event_type", event.EventType),
		zap.String("order_id", event.OrderID),
		zap.String("payment_key", event.PaymentKey),
		zap.String("status", event.Status),
		zap.Any("event_data", event.Data))

	// Handle different event types
	switch event.Status {
	case "DONE": // Payment completed
		err = h.handlePaymentCompleted(ctx, event)
	case "CANCELED": // Payment cancelled
		err = h.handlePaymentCancelled(ctx, event)
	case "PARTIAL_CANCELED": // Partial refund
		err = h.handlePaymentRefunded(ctx, event)
	case "EXPIRED", "ABORTED": // Payment failed
		err = h.handlePaymentFailed(ctx, event)
	default:
		h.logger.Warn("Unknown webhook event status",
			zap.String("status", event.Status),
			zap.String("order_id", event.OrderID))
	}

	if err != nil {
		h.logger.Error("Failed to handle webhook event",
			zap.String("event_type", event.EventType),
			zap.String("order_id", event.OrderID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to process webhook event",
			"code":  "WEBHOOK_HANDLER_ERROR",
		})
	}

	// Return success response
	return c.JSON(http.StatusOK, echo.Map{
		"status": "ok",
	})
}

// handlePaymentCompleted handles successful payment events
func (h *TossWebhookHandler) handlePaymentCompleted(ctx context.Context, event *provider.WebhookEvent) error {
	if event.OrderID == "" {
		h.logger.Warn("Missing order ID in webhook event")
		return nil
	}

	// Get payment by order ID
	payment, err := h.paymentRepo.GetByOrderID(ctx, event.OrderID)
	if err != nil {
		return err
	}

	if payment == nil {
		h.logger.Warn("Payment not found for order ID",
			zap.String("order_id", event.OrderID))
		return nil
	}

	// Skip if already completed
	if payment.Status == entity.PaymentStatusCompleted {
		h.logger.Info("Payment already completed",
			zap.String("order_id", event.OrderID))
		return nil
	}

	// Update payment status
	updates := map[string]interface{}{
		"status":                     string(entity.PaymentStatusCompleted),
		"provider_payment_intent_id": event.PaymentKey,
		"provider_charge_id":         event.TransactionKey,
		"paid_at":                    gorm.Expr("NOW()"),
		"provider_payment_data":      event.Data,
	}

	err = h.paymentRepo.UpdatePaymentAfterConfirm(ctx, event.OrderID, updates)
	if err != nil {
		return err
	}

	h.logger.Info("Payment marked as completed via webhook",
		zap.String("order_id", event.OrderID),
		zap.String("payment_key", event.PaymentKey))

	// TODO: Allocate credits if applicable
	// This would require checking if the payment has associated credits to allocate

	return nil
}

// handlePaymentCancelled handles cancelled payment events
func (h *TossWebhookHandler) handlePaymentCancelled(ctx context.Context, event *provider.WebhookEvent) error {
	if event.OrderID == "" {
		h.logger.Warn("Missing order ID in webhook event")
		return nil
	}

	// Update payment status
	updates := map[string]interface{}{
		"status":                string(entity.PaymentStatusCanceled),
		"provider_payment_data": event.Data,
	}

	// Extract cancellation details if available
	if cancelData, ok := event.Data["cancels"].([]interface{}); ok && len(cancelData) > 0 {
		if firstCancel, ok := cancelData[0].(map[string]interface{}); ok {
			if cancelReason, ok := firstCancel["cancelReason"].(string); ok {
				updates["failure_message"] = cancelReason
			}
			if canceledAt, ok := firstCancel["canceledAt"].(string); ok {
				if t, err := time.Parse(time.RFC3339, canceledAt); err == nil {
					updates["updated_at"] = t
				}
			}
		}
	}

	err := h.paymentRepo.UpdatePaymentAfterConfirm(ctx, event.OrderID, updates)
	if err != nil {
		return err
	}

	h.logger.Info("Payment marked as cancelled via webhook",
		zap.String("order_id", event.OrderID))

	return nil
}

// handlePaymentRefunded handles refunded payment events
func (h *TossWebhookHandler) handlePaymentRefunded(ctx context.Context, event *provider.WebhookEvent) error {
	if event.OrderID == "" {
		h.logger.Warn("Missing order ID in webhook event")
		return nil
	}

	// Update payment status
	updates := map[string]interface{}{
		"status":                string(entity.PaymentStatusRefunded),
		"provider_payment_data": event.Data,
	}

	err := h.paymentRepo.UpdatePaymentAfterConfirm(ctx, event.OrderID, updates)
	if err != nil {
		return err
	}

	h.logger.Info("Payment marked as refunded via webhook",
		zap.String("order_id", event.OrderID))

	// TODO: Handle credit deduction if applicable

	return nil
}

// handlePaymentFailed handles failed payment events
func (h *TossWebhookHandler) handlePaymentFailed(ctx context.Context, event *provider.WebhookEvent) error {
	if event.OrderID == "" {
		h.logger.Warn("Missing order ID in webhook event")
		return nil
	}

	// Extract failure reason
	var failureMessage string
	failureCode := event.Status
	if msg, ok := event.Data["message"].(string); ok {
		failureMessage = msg
	}
	if failureMessage == "" && event.Status == "EXPIRED" {
		failureMessage = "Payment expired"
	}
	if failureMessage == "" && event.Status == "ABORTED" {
		failureMessage = "Payment aborted"
	}
	if failureData, ok := event.Data["failure"].(map[string]interface{}); ok {
		if code, ok := failureData["code"].(string); ok && code != "" {
			failureCode = code
		}
		if msg, ok := failureData["message"].(string); ok && msg != "" {
			failureMessage = msg
		}
	}

	// Update payment status
	updates := map[string]interface{}{
		"status":                string(entity.PaymentStatusFailed),
		"failure_code":          failureCode,
		"failure_message":       failureMessage,
		"provider_payment_data": event.Data,
	}

	err := h.paymentRepo.UpdatePaymentAfterConfirm(ctx, event.OrderID, updates)
	if err != nil {
		return err
	}

	h.logger.Info("Payment marked as failed via webhook",
		zap.String("order_id", event.OrderID),
		zap.String("status", event.Status),
		zap.String("failure_message", failureMessage))

	return nil
}

// TossWebhookPayload represents the structure of Toss webhook payload
type TossWebhookPayload struct {
	EventType string               `json:"eventType"`
	CreatedAt string               `json:"createdAt"`
	Data      TossWebhookEventData `json:"data"`
}

// TossWebhookEventData represents the nested Toss event payload
type TossWebhookEventData struct {
	OrderID       string      `json:"orderId"`
	PaymentKey    string      `json:"paymentKey"`
	Status        string      `json:"status"`
	Failure       interface{} `json:"failure,omitempty"`
	Cancels       interface{} `json:"cancels,omitempty"`
	TransactionID string      `json:"transactionId,omitempty"`
}
