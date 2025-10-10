package stripe

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	"go.uber.org/zap"
)

// StripeProvider implements the PaymentProvider interface for Stripe (stub implementation)
type StripeProvider struct {
	secretKey string
	logger    *zap.Logger
}

// NewStripeProvider creates a new Stripe provider (stub)
func NewStripeProvider(secretKey string, logger *zap.Logger) *StripeProvider {
	return &StripeProvider{
		secretKey: secretKey,
		logger:    logger,
	}
}

// GetProviderName returns the provider name
func (s *StripeProvider) GetProviderName() string {
	return string(provider.ProviderTypeStripe)
}

// InitializePayment creates a new payment intent with Stripe (stub)
func (s *StripeProvider) InitializePayment(ctx context.Context, req *provider.InitializePaymentRequest) (*provider.InitializePaymentResponse, error) {
	s.logger.Warn("StripeProvider: InitializePayment not implemented",
		zap.String("order_id", req.OrderID))

	return nil, &provider.ProviderError{
		Code:    "NOT_IMPLEMENTED",
		Message: "Stripe one-time payment is not yet implemented",
	}
}

// ConfirmPayment confirms and captures a payment with Stripe (stub)
func (s *StripeProvider) ConfirmPayment(ctx context.Context, req *provider.ConfirmPaymentRequest) (*provider.ConfirmPaymentResponse, error) {
	s.logger.Warn("StripeProvider: ConfirmPayment not implemented",
		zap.String("order_id", req.OrderID))

	return nil, &provider.ProviderError{
		Code:    "NOT_IMPLEMENTED",
		Message: "Stripe one-time payment confirmation is not yet implemented",
	}
}

// HandleWebhook processes Stripe webhook events (stub)
func (s *StripeProvider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*provider.WebhookEvent, error) {
	s.logger.Warn("StripeProvider: HandleWebhook not implemented")

	return nil, &provider.ProviderError{
		Code:    "NOT_IMPLEMENTED",
		Message: "Stripe webhook handling is not yet implemented",
	}
}