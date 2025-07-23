package http

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	logger        *zap.Logger
	webhookSecret string
	subscriptions map[string]*entity.Subscription
	payments      []PaymentData
	mu            sync.RWMutex
}

type PaymentData struct {
	InvoiceID      string
	CustomerID     string
	SubscriptionID string
	Amount         int64
	Status         string
	CreatedAt      time.Time
}

func NewWebhookHandler(logger *zap.Logger, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		logger:        logger,
		webhookSecret: webhookSecret,
		subscriptions: make(map[string]*entity.Subscription),
		payments:      make([]PaymentData, 0),
	}
}

func (h *WebhookHandler) GetWebhookData(c echo.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	return c.JSON(http.StatusOK, echo.Map{
		"subscriptions": h.subscriptions,
		"payments":      h.payments,
		"payment_count": len(h.payments),
	})
}

func (h *WebhookHandler) HandleWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Error("Error reading request body", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error reading request body"})
	}
	
	sig := c.Request().Header.Get("Stripe-Signature")
	
	event, err := webhook.ConstructEventWithOptions(
		body, 
		sig, 
		h.webhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	
	if err != nil {
		h.logger.Error("Webhook signature verification failed", 
			zap.Error(err),
			zap.String("signature", sig),
		)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Webhook signature verification failed: " + err.Error(),
		})
	}
	
	h.logger.Info("Webhook Event Received",
		zap.String("type", string(event.Type)),
		zap.String("id", event.ID),
		zap.Time("created", time.Unix(event.Created, 0)),
	)
	
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			h.logger.Error("Error parsing checkout session", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}
		
		h.logger.Info("CHECKOUT SESSION COMPLETED",
			zap.String("session_id", session.ID),
			zap.String("customer_email", session.CustomerEmail),
			zap.String("payment_status", string(session.PaymentStatus)),
		)
		
		if session.Mode == "subscription" {
			if session.Customer != nil && session.Customer.ID != "" {
				h.mu.Lock()
				h.subscriptions[session.Customer.ID] = &entity.Subscription{
					CustomerID:    session.Customer.ID,
					CustomerEmail: session.CustomerEmail,
					Status:        "pending",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				h.mu.Unlock()
				
				h.logger.Info("Temporary subscription data saved",
					zap.String("customer_id", session.Customer.ID),
					zap.String("email", session.CustomerEmail),
				)
			}
		}
		
	case stripe.EventTypeCustomerSubscriptionCreated, stripe.EventTypeCustomerSubscriptionUpdated:
		var rawData map[string]interface{}
		if err := json.Unmarshal(event.Data.Raw, &rawData); err != nil {
			h.logger.Error("Error parsing raw subscription data", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}
		
		subscriptionID, _ := rawData["id"].(string)
		status, _ := rawData["status"].(string)
		customerID, _ := rawData["customer"].(string)
		
		currentPeriodEnd := int64(0)
		if cpe, ok := rawData["current_period_end"].(float64); ok {
			currentPeriodEnd = int64(cpe)
		}
		
		h.logger.Info("SUBSCRIPTION CREATED/UPDATED",
			zap.String("subscription_id", subscriptionID),
			zap.String("customer_id", customerID),
			zap.String("status", status),
			zap.Time("period_end", time.Unix(currentPeriodEnd, 0)),
		)
		
		if customerID != "" {
			h.mu.Lock()
			
			if existing, ok := h.subscriptions[customerID]; ok {
				existing.ID = subscriptionID
				existing.Status = status
				existing.CurrentPeriodEnd = time.Unix(currentPeriodEnd, 0)
				existing.UpdatedAt = time.Now()
			} else {
				h.subscriptions[customerID] = &entity.Subscription{
					ID:               subscriptionID,
					CustomerID:       customerID,
					Status:           status,
					CurrentPeriodEnd: time.Unix(currentPeriodEnd, 0),
					CreatedAt:        time.Now(),
					UpdatedAt:        time.Now(),
				}
			}
			
			h.logger.Info("Subscription data saved",
				zap.String("customer_id", customerID),
				zap.String("subscription_id", subscriptionID),
				zap.Time("period_end", time.Unix(currentPeriodEnd, 0)),
			)
			
			h.mu.Unlock()
		}
		
	case stripe.EventTypeCustomerSubscriptionDeleted:
		var rawData map[string]interface{}
		if err := json.Unmarshal(event.Data.Raw, &rawData); err != nil {
			h.logger.Error("Error parsing subscription deletion", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}
		
		customerID, _ := rawData["customer"].(string)
		
		h.logger.Info("SUBSCRIPTION DELETED",
			zap.String("customer_id", customerID),
		)
		
		if customerID != "" {
			h.mu.Lock()
			if sub, exists := h.subscriptions[customerID]; exists {
				sub.Status = "canceled"
				sub.UpdatedAt = time.Now()
			}
			h.mu.Unlock()
		}
	
	case stripe.EventTypeInvoicePaymentSucceeded:
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			h.logger.Error("Error parsing invoice", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}
		
		h.logger.Info("PAYMENT SUCCEEDED",
			zap.String("invoice_id", invoice.ID),
			zap.Int64("amount_paid", invoice.AmountPaid),
		)
		
		h.mu.Lock()
		payment := PaymentData{
			InvoiceID:  invoice.ID,
			Amount:     invoice.AmountPaid,
			Status:     "succeeded",
			CreatedAt:  time.Now(),
		}
		
		if invoice.Customer != nil {
			payment.CustomerID = invoice.Customer.ID
		}
		
		if invoice.Subscription != nil {
			payment.SubscriptionID = invoice.Subscription.ID
		}
		
		h.payments = append(h.payments, payment)
		
		if payment.CustomerID != "" {
			if sub, exists := h.subscriptions[payment.CustomerID]; exists {
				if invoice.Lines != nil && len(invoice.Lines.Data) > 0 {
					line := invoice.Lines.Data[0]
					if line.Period != nil && line.Period.End > 0 {
						sub.CurrentPeriodEnd = time.Unix(line.Period.End, 0)
						sub.UpdatedAt = time.Now()
						h.logger.Info("Subscription period extended",
							zap.Time("new_period_end", sub.CurrentPeriodEnd),
						)
					}
				}
			}
		}
		h.mu.Unlock()    

	case stripe.EventTypeInvoicePaymentFailed:
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			h.logger.Error("Error parsing invoice", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}
		
		h.logger.Warn("PAYMENT FAILED",
			zap.String("invoice_id", invoice.ID),
			zap.Int64("amount_due", invoice.AmountDue),
		)
		
		h.mu.Lock()
		payment := PaymentData{
			InvoiceID:  invoice.ID,
			Amount:     invoice.AmountDue,
			Status:     "failed",
			CreatedAt:  time.Now(),
		}
		
		if invoice.Customer != nil {
			payment.CustomerID = invoice.Customer.ID
		}
		
		h.payments = append(h.payments, payment)
		h.mu.Unlock()
		
	default:
		h.logger.Warn("Unhandled event type",
			zap.String("type", string(event.Type)),
		)
	}
	
	return c.JSON(http.StatusOK, echo.Map{"received": true})
}