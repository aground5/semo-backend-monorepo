package usecase

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/product"
	"github.com/stripe/stripe-go/v79/subscription"
	domainErrors "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

// SubscriptionService handles subscription-related business logic with Stripe integration
type SubscriptionService struct {
	customerMappingRepo repository.CustomerMappingRepository
	subscriptionRepo    repository.SubscriptionRepository
	logger              *zap.Logger
}

// NewSubscriptionService creates a new subscription service instance
func NewSubscriptionService(
	customerMappingRepo repository.CustomerMappingRepository,
	subscriptionRepo repository.SubscriptionRepository,
	logger *zap.Logger,
) *SubscriptionService {
	return &SubscriptionService{
		customerMappingRepo: customerMappingRepo,
		subscriptionRepo:    subscriptionRepo,
		logger:              logger,
	}
}

// GetActiveSubscriptionForUser finds the active subscription for a given user ID
func (s *SubscriptionService) GetActiveSubscriptionForUser(ctx context.Context, userID string) (*stripe.Subscription, error) {
	// Look up customer mapping
	customerMapping, err := s.customerMappingRepo.GetByUniversalID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer mapping: %w", err)
	}

	if customerMapping == nil {
		return nil, domainErrors.ErrNoCustomerMapping
	}

	// Find active subscription
	return s.GetActiveSubscriptionForCustomer(ctx, customerMapping.StripeCustomerID)
}

// GetActiveSubscriptionForCustomer finds the active subscription for a given customer ID
func (s *SubscriptionService) GetActiveSubscriptionForCustomer(ctx context.Context, customerID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String("all"),
	}
	// Expand only up to price level (4 levels max)
	params.AddExpand("data.items.data.price")

	iter := subscription.List(params)

	var activeSub *stripe.Subscription
	for iter.Next() {
		sub := iter.Subscription()
		if sub.Status == "active" || sub.Status == "trialing" {
			activeSub = sub
			break
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("error listing subscriptions: %w", err)
	}

	if activeSub == nil {
		return nil, domainErrors.ErrNoActiveSubscription
	}

	// Fetch product details for each subscription item if needed
	for _, item := range activeSub.Items.Data {
		if item.Price != nil && item.Price.Product != nil && item.Price.Product.ID != "" {
			// Product is already expanded as an ID, fetch full product details
			prod, err := product.Get(item.Price.Product.ID, nil)
			if err != nil {
				s.logger.Warn("Failed to fetch product details",
					zap.String("product_id", item.Price.Product.ID),
					zap.Error(err))
				// Continue without product details
			} else {
				// Replace the product reference with full product data
				item.Price.Product = prod
			}
		}
	}

	return activeSub, nil
}

// CancelSubscriptionForUser cancels the active subscription for a given user ID
func (s *SubscriptionService) CancelSubscriptionForUser(ctx context.Context, userID string) (*stripe.Subscription, error) {
	// Get the active subscription
	activeSub, err := s.GetActiveSubscriptionForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cancel the subscription at period end
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	// Expand the response to include price details (but not beyond 4 levels)
	params.AddExpand("items.data.price")
	
	updatedSub, err := subscription.Update(activeSub.ID, params)

	if err != nil {
		return nil, fmt.Errorf("failed to cancel subscription: %w", err)
	}

	s.logger.Info("Subscription canceled successfully",
		zap.String("subscription_id", updatedSub.ID),
		zap.String("user_id", userID),
		zap.Bool("cancel_at_period_end", updatedSub.CancelAtPeriodEnd),
	)

	// Fetch product details for each subscription item if needed
	for _, item := range updatedSub.Items.Data {
		if item.Price != nil && item.Price.Product != nil && item.Price.Product.ID != "" {
			// Product is already expanded as an ID, fetch full product details
			prod, err := product.Get(item.Price.Product.ID, nil)
			if err != nil {
				s.logger.Warn("Failed to fetch product details",
					zap.String("product_id", item.Price.Product.ID),
					zap.Error(err))
				// Continue without product details
			} else {
				// Replace the product reference with full product data
				item.Price.Product = prod
			}
		}
	}

	return updatedSub, nil
}
