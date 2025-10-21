package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	tossProvider "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider/toss"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TossWebhookHandler handles TossPayments webhook events
type TossWebhookHandler struct {
	logger         *zap.Logger
	paymentRepo    repository.PaymentRepository
	creditService  *usecase.CreditService
	tossProvider   *tossProvider.TossProvider
	supabaseSecret string
}

// NewTossWebhookHandler creates a new TossWebhookHandler instance
func NewTossWebhookHandler(
	logger *zap.Logger,
	paymentRepo repository.PaymentRepository,
	creditService *usecase.CreditService,
	secretKey string,
	clientKey string,
	supabaseSecret string,
) *TossWebhookHandler {
	return &TossWebhookHandler{
		logger:         logger,
		paymentRepo:    paymentRepo,
		creditService:  creditService,
		tossProvider:   tossProvider.NewTossProvider(secretKey, clientKey, logger),
		supabaseSecret: supabaseSecret,
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
	xWebhookSecret := c.Request().Header.Get("X-Webhook-Secret")

	if h.supabaseSecret != "" && xWebhookSecret == h.supabaseSecret {
		var supabasePayload SupabaseWebhookPayload
		if err := json.Unmarshal(body, &supabasePayload); err != nil {
			h.logger.Warn("Supabase webhook payload parse failed",
				zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": "Failed to parse Supabase webhook payload",
				"code":  "INVALID_SUPABASE_PAYLOAD",
			})
		}

		h.logger.Info("Received Supabase webhook event",
			zap.String("user_id", supabasePayload.UserID),
			zap.String("email", supabasePayload.Email),
			zap.String("service_provider", supabasePayload.ServiceProvider),
			zap.String("confirmed_at", supabasePayload.ConfirmedAt),
			zap.String("created_at", supabasePayload.CreatedAt))

		if h.creditService == nil {
			h.logger.Warn("Credit service not configured; skipping Supabase credit allocation",
				zap.String("user_id", supabasePayload.UserID))
			return c.JSON(http.StatusOK, echo.Map{
				"status": "ok",
			})
		}

		if supabasePayload.UserID == "" || supabasePayload.ServiceProvider == "" {
			h.logger.Warn("Supabase payload missing required fields",
				zap.String("user_id", supabasePayload.UserID),
				zap.String("service_provider", supabasePayload.ServiceProvider))
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": "Missing required Supabase payload fields",
				"code":  "INVALID_SUPABASE_PAYLOAD",
			})
		}

		userUUID, err := uuid.Parse(supabasePayload.UserID)
		if err != nil {
			h.logger.Warn("Invalid Supabase user ID; skipping credit allocation",
				zap.String("user_id", supabasePayload.UserID),
				zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": "Invalid user ID format",
				"code":  "INVALID_SUPABASE_PAYLOAD",
			})
		}

		referenceID := fmt.Sprintf("supabase:%s:%s:%s", supabasePayload.ServiceProvider, supabasePayload.UserID, supabasePayload.ReferenceTimestamp())
		description := fmt.Sprintf("Supabase confirmation credit for %s", supabasePayload.Email)

		if _, _, err := h.creditService.AllocateCreditsManual(c.Request().Context(), userUUID, supabasePayload.ServiceProvider, 1, description, referenceID); err != nil {
			h.logger.Error("Failed to allocate Supabase confirmation credit",
				zap.String("user_id", supabasePayload.UserID),
				zap.String("service_provider", supabasePayload.ServiceProvider),
				zap.Error(err))
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "Failed to allocate credit",
				"code":  "SUPABASE_CREDIT_ALLOCATION_FAILED",
			})
		}

		h.logger.Info("Supabase confirmation credit allocated",
			zap.String("user_id", supabasePayload.UserID),
			zap.String("service_provider", supabasePayload.ServiceProvider))

		return c.JSON(http.StatusOK, echo.Map{
			"status": "ok",
		})
	}

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

	alreadyCompleted := payment.Status == entity.PaymentStatusCompleted

	updates := map[string]interface{}{
		"provider_payment_data": event.Data,
	}

	if !alreadyCompleted {
		updates["status"] = string(entity.PaymentStatusCompleted)
		updates["provider_payment_intent_id"] = event.PaymentKey
		updates["provider_charge_id"] = event.TransactionKey
		updates["paid_at"] = gorm.Expr("NOW()")
	}

	if err := h.paymentRepo.UpdatePaymentAfterConfirm(ctx, event.OrderID, updates); err != nil {
		return err
	}

	if alreadyCompleted {
		h.logger.Info("Payment already marked completed, ensured provider data is synced and proceeding with credit allocation",
			zap.String("order_id", event.OrderID),
			zap.String("payment_key", event.PaymentKey))
	} else {
		h.logger.Info("Payment marked as completed via webhook",
			zap.String("order_id", event.OrderID),
			zap.String("payment_key", event.PaymentKey))
	}

	if h.creditService == nil {
		h.logger.Warn("Credit service not configured; skipping credit allocation",
			zap.String("order_id", event.OrderID))
		return nil
	}

	if payment.UniversalID == "" {
		h.logger.Warn("Cannot allocate credits without universal ID",
			zap.String("order_id", event.OrderID))
		return nil
	}

	universalUUID, err := uuid.Parse(payment.UniversalID)
	if err != nil {
		h.logger.Error("Invalid universal ID on payment; skipping credit allocation",
			zap.String("order_id", event.OrderID),
			zap.String("universal_id", payment.UniversalID),
			zap.Error(err))
		return nil
	}

	extractString := func(m map[string]interface{}, key string) string {
		if m == nil {
			return ""
		}
		if val, ok := m[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
		return ""
	}

	var metadata map[string]interface{}
	var planID string

	candidateSources := []map[string]interface{}{}
	if event.Data != nil {
		if md, ok := event.Data["metadata"].(map[string]interface{}); ok {
			candidateSources = append(candidateSources, md)
		}
		candidateSources = append(candidateSources, event.Data)
	}
	if payment.Metadata != nil {
		if md, ok := payment.Metadata["metadata"].(map[string]interface{}); ok {
			candidateSources = append(candidateSources, md)
		}
		candidateSources = append(candidateSources, payment.Metadata)
	}

	for _, candidate := range candidateSources {
		if candidate == nil {
			continue
		}
		if planID == "" {
			if value := extractString(candidate, "plan_id"); value != "" {
				planID = value
				metadata = candidate
				break
			}
			if value := extractString(candidate, "planId"); value != "" {
				planID = value
				metadata = candidate
				break
			}
		}
	}

	if metadata == nil && payment.Metadata != nil {
		metadata = payment.Metadata
	}

	if planID == "" {
		h.logger.Warn("No plan_id found in Toss metadata; skipping credit allocation",
			zap.String("order_id", event.OrderID),
			zap.Any("metadata_keys", func() []string {
				keys := make([]string, 0, len(metadata))
				for k := range metadata {
					keys = append(keys, k)
				}
				return keys
			}()))
		return nil
	}

	customerKey := extractString(metadata, "customer_key")
	serviceProvider := extractString(metadata, "service_provider")
	h.logger.Info("Attempting credit allocation from Toss metadata",
		zap.String("order_id", event.OrderID),
		zap.String("universal_id", payment.UniversalID),
		zap.String("plan_id", planID),
		zap.String("customer_key", customerKey),
		zap.String("service_provider", serviceProvider))

	allocatedCredits, err := h.creditService.AllocateCreditsForPayment(
		ctx,
		universalUUID,
		event.OrderID,
		"",
		planID,
		serviceProvider,
	)
	if err != nil {
		h.logger.Error("Failed to allocate credits from Toss webhook",
			zap.String("order_id", event.OrderID),
			zap.String("universal_id", payment.UniversalID),
			zap.String("plan_id", planID),
			zap.Error(err))
		return nil
	}

	if allocatedCredits == 0 {
		h.logger.Info("No new credits allocated for Toss webhook event (likely already processed)",
			zap.String("order_id", event.OrderID),
			zap.String("universal_id", payment.UniversalID),
			zap.String("plan_id", planID))
		return nil
	}

	h.logger.Info("Credits allocated successfully from Toss webhook",
		zap.String("order_id", event.OrderID),
		zap.String("universal_id", payment.UniversalID),
		zap.String("plan_id", planID),
		zap.Int("credits", allocatedCredits))

	creditUpdates := map[string]interface{}{
		"credits_allocated":    decimal.NewFromInt(int64(allocatedCredits)),
		"credits_allocated_at": gorm.Expr("NOW()"),
	}

	if err := h.paymentRepo.UpdatePaymentAfterConfirm(ctx, event.OrderID, creditUpdates); err != nil {
		h.logger.Error("Failed to update payment after credit allocation",
			zap.String("order_id", event.OrderID),
			zap.String("universal_id", payment.UniversalID),
			zap.Int("credits", allocatedCredits),
			zap.Error(err))
	} else {
		h.logger.Info("Payment record updated with credit allocation",
			zap.String("order_id", event.OrderID),
			zap.Int("credits_allocated", allocatedCredits))
	}

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

// SupabaseWebhookPayload captures the Supabase signup webhook payload
type SupabaseWebhookPayload struct {
	ConfirmedAt     string `json:"confirmed_at"`
	CreatedAt       string `json:"created_at"`
	Email           string `json:"email"`
	ServiceProvider string `json:"service_provider"`
	UserID          string `json:"user_id"`
}

// ReferenceTimestamp returns the timestamp used for idempotency reference
func (p SupabaseWebhookPayload) ReferenceTimestamp() string {
	if p.ConfirmedAt != "" {
		return p.ConfirmedAt
	}
	return p.CreatedAt
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
