package provider

import (
	"context"
	"time"
)

// PaymentProvider defines the interface for payment providers (Stripe, Toss, etc.)
type PaymentProvider interface {
	// InitializePayment creates a new payment intent/order
	InitializePayment(ctx context.Context, req *InitializePaymentRequest) (*InitializePaymentResponse, error)

	// ConfirmPayment confirms and captures a payment
	ConfirmPayment(ctx context.Context, req *ConfirmPaymentRequest) (*ConfirmPaymentResponse, error)

	// HandleWebhook processes provider-specific webhook events
	HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)

	// GetProviderName returns the provider name
	GetProviderName() string
}

// InitializePaymentRequest represents a provider-agnostic payment initialization request
type InitializePaymentRequest struct {
	UniversalID  string                 `json:"universal_id"`
	Amount       int64                  `json:"amount"`        // Amount in smallest currency unit
	Currency     string                 `json:"currency"`
	OrderID      string                 `json:"order_id"`      // Internal order ID
	OrderName    string                 `json:"order_name"`    // Description
	CustomerKey  string                 `json:"customer_key"`  // Customer identifier
	PlanID       string                 `json:"plan_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// InitializePaymentResponse represents the response from payment initialization
type InitializePaymentResponse struct {
	OrderID           string                 `json:"order_id"`
	PaymentKey        string                 `json:"payment_key,omitempty"`        // Provider payment ID
	ClientSecret      string                 `json:"client_secret,omitempty"`      // For client-side confirmation
	Status            string                 `json:"status"`
	Amount            int64                  `json:"amount"`
	Currency          string                 `json:"currency"`
	ProviderData      map[string]interface{} `json:"provider_data,omitempty"`      // Provider-specific data
}

// ConfirmPaymentRequest represents a payment confirmation request
type ConfirmPaymentRequest struct {
	OrderID      string                 `json:"order_id"`
	PaymentKey   string                 `json:"payment_key"`   // Provider payment ID
	Amount       int64                  `json:"amount"`
	ProviderData map[string]interface{} `json:"provider_data,omitempty"` // Provider-specific data
}

// ConfirmPaymentResponse represents the response from payment confirmation
type ConfirmPaymentResponse struct {
	OrderID         string                 `json:"order_id"`
	PaymentKey      string                 `json:"payment_key"`
	TransactionKey  string                 `json:"transaction_key,omitempty"` // Provider transaction ID
	Status          PaymentStatus          `json:"status"`
	Amount          int64                  `json:"amount"`
	Currency        string                 `json:"currency"`
	PaymentMethod   string                 `json:"payment_method,omitempty"`
	PaidAt          *time.Time             `json:"paid_at,omitempty"`
	ProviderData    map[string]interface{} `json:"provider_data,omitempty"`
}

// WebhookEvent represents a provider webhook event
type WebhookEvent struct {
	EventID        string                 `json:"event_id"`
	EventType      string                 `json:"event_type"`
	OrderID        string                 `json:"order_id,omitempty"`
	PaymentKey     string                 `json:"payment_key,omitempty"`
	TransactionKey string                 `json:"transaction_key,omitempty"`
	Status         string                 `json:"status"`
	Amount         int64                  `json:"amount,omitempty"`
	Data           map[string]interface{} `json:"data"`
	CreatedAt      time.Time              `json:"created_at"`
}

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// ProviderType represents the type of payment provider
type ProviderType string

const (
	ProviderTypeStripe ProviderType = "stripe"
	ProviderTypeToss   ProviderType = "toss"
)

// Error types for provider operations
type ProviderError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ProviderError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}