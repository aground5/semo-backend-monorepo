package provider

import (
	"fmt"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	stripeProvider "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider/stripe"
	tossProvider "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider/toss"
	"go.uber.org/zap"
)

// Factory creates payment providers based on the provider type
type Factory struct {
	config *config.Config
	logger *zap.Logger
}

// NewFactory creates a new provider factory
func NewFactory(config *config.Config, logger *zap.Logger) *Factory {
	return &Factory{
		config: config,
		logger: logger,
	}
}

// GetProvider returns a payment provider based on the provider type
func (f *Factory) GetProvider(providerType provider.ProviderType) (provider.PaymentProvider, error) {
	switch providerType {
	case provider.ProviderTypeToss:
		return f.createTossProvider()
	case provider.ProviderTypeStripe:
		return f.createStripeProvider()
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetProviderFromString returns a payment provider from a string type
func (f *Factory) GetProviderFromString(providerStr string) (provider.PaymentProvider, error) {
	// Default to Toss if not specified
	if providerStr == "" {
		providerStr = string(provider.ProviderTypeToss)
	}

	providerType := provider.ProviderType(providerStr)
	return f.GetProvider(providerType)
}

// createTossProvider creates a new Toss provider instance
func (f *Factory) createTossProvider() (provider.PaymentProvider, error) {
	if f.config.Service.Toss.SecretKey == "" {
		return nil, fmt.Errorf("Toss secret key not configured")
	}

	return tossProvider.NewTossProvider(
		f.config.Service.Toss.SecretKey,
		f.config.Service.Toss.ClientKey,
		f.logger,
	), nil
}

// createStripeProvider creates a new Stripe provider instance
func (f *Factory) createStripeProvider() (provider.PaymentProvider, error) {
	if f.config.Service.StripeSecretKey == "" {
		return nil, fmt.Errorf("Stripe secret key not configured")
	}

	return stripeProvider.NewStripeProvider(
		f.config.Service.StripeSecretKey,
		f.logger,
	), nil
}