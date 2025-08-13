package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/price"
	"github.com/stripe/stripe-go/v79/product"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"go.uber.org/zap"
)

// PlanSyncService handles synchronization of Stripe products/prices with local database
type PlanSyncService struct {
	planRepo repository.PlanRepository
	logger   *zap.Logger
}

// NewPlanSyncService creates a new plan synchronization service
func NewPlanSyncService(planRepo repository.PlanRepository, logger *zap.Logger) *PlanSyncService {
	return &PlanSyncService{
		planRepo: planRepo,
		logger:   logger,
	}
}

// SyncProductEvent handles product-related webhook events
func (s *PlanSyncService) SyncProductEvent(ctx context.Context, eventType string, eventData json.RawMessage) error {
	switch eventType {
	case "product.created", "product.updated":
		return s.handleProductUpsert(ctx, eventData)
	case "product.deleted":
		return s.handleProductDeleted(ctx, eventData)
	default:
		return fmt.Errorf("unhandled product event type: %s", eventType)
	}
}

// SyncPriceEvent handles price-related webhook events
func (s *PlanSyncService) SyncPriceEvent(ctx context.Context, eventType string, eventData json.RawMessage) error {
	switch eventType {
	case "price.created", "price.updated":
		return s.handlePriceUpsert(ctx, eventData)
	case "price.deleted":
		return s.handlePriceDeleted(ctx, eventData)
	default:
		return fmt.Errorf("unhandled price event type: %s", eventType)
	}
}

// handleProductUpsert handles product creation/update
func (s *PlanSyncService) handleProductUpsert(ctx context.Context, eventData json.RawMessage) error {
	var prod stripe.Product
	if err := json.Unmarshal(eventData, &prod); err != nil {
		return fmt.Errorf("failed to unmarshal product data: %w", err)
	}

	s.logger.Info("Syncing product",
		zap.String("product_id", prod.ID),
		zap.String("name", prod.Name))

	// Get all prices for this product
	params := &stripe.PriceListParams{
		Product: stripe.String(prod.ID),
		Active:  stripe.Bool(true),
	}

	iter := price.List(params)
	for iter.Next() {
		p := iter.Price()
		if err := s.SyncPriceWithProduct(ctx, p, &prod); err != nil {
			s.logger.Error("Failed to sync price",
				zap.String("price_id", p.ID),
				zap.Error(err))
			// Continue with other prices
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("error listing prices: %w", err)
	}

	return nil
}

// handleProductDeleted handles product deletion
func (s *PlanSyncService) handleProductDeleted(ctx context.Context, eventData json.RawMessage) error {
	var prod stripe.Product
	if err := json.Unmarshal(eventData, &prod); err != nil {
		return fmt.Errorf("failed to unmarshal product data: %w", err)
	}

	s.logger.Info("Deactivating plans for deleted product",
		zap.String("product_id", prod.ID))

	// Get all plans for this product
	plans, err := s.planRepo.GetByProductID(ctx, prod.ID)
	if err != nil {
		return fmt.Errorf("failed to get plans for product: %w", err)
	}

	// Deactivate all related plans
	for _, plan := range plans {
		if err := s.planRepo.Delete(ctx, plan.StripePriceID); err != nil {
			s.logger.Error("Failed to deactivate plan",
				zap.String("price_id", plan.StripePriceID),
				zap.Error(err))
			// Continue with other plans
		}
	}

	return nil
}

// handlePriceUpsert handles price creation/update
func (s *PlanSyncService) handlePriceUpsert(ctx context.Context, eventData json.RawMessage) error {
	var p stripe.Price
	if err := json.Unmarshal(eventData, &p); err != nil {
		return fmt.Errorf("failed to unmarshal price data: %w", err)
	}

	s.logger.Info("Syncing price",
		zap.String("price_id", p.ID),
		zap.String("product_id", p.Product.ID))

	// Get full product details
	prod, err := product.Get(p.Product.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to get product details: %w", err)
	}

	return s.SyncPriceWithProduct(ctx, &p, prod)
}

// handlePriceDeleted handles price deletion
func (s *PlanSyncService) handlePriceDeleted(ctx context.Context, eventData json.RawMessage) error {
	var p stripe.Price
	if err := json.Unmarshal(eventData, &p); err != nil {
		return fmt.Errorf("failed to unmarshal price data: %w", err)
	}

	s.logger.Info("Deactivating plan for deleted price",
		zap.String("price_id", p.ID))

	return s.planRepo.Delete(ctx, p.ID)
}

// SyncPriceWithProduct syncs a price with its product information
func (s *PlanSyncService) SyncPriceWithProduct(ctx context.Context, p *stripe.Price, prod *stripe.Product) error {
	// Determine plan type based on price type
	var planType string
	if p.Type == stripe.PriceTypeRecurring && p.Recurring != nil {
		planType = model.PlanTypeSubscription
	} else if p.Type == stripe.PriceTypeOneTime {
		planType = model.PlanTypeOneTime
	} else {
		s.logger.Debug("Skipping unsupported price type",
			zap.String("price_id", p.ID),
			zap.String("price_type", string(p.Type)))
		return nil
	}

	// Extract credits from product metadata or default
	creditsPerCycle := 100 // Default
	if credits, ok := prod.Metadata["credits_per_cycle"]; ok {
		fmt.Sscanf(credits, "%d", &creditsPerCycle)
	}

	// Extract features from product metadata
	features := make(model.Features)
	if featuresJSON, ok := prod.Metadata["features"]; ok {
		json.Unmarshal([]byte(featuresJSON), &features)
	}

	// Add price information to features
	features["amount"] = p.UnitAmount
	features["currency"] = string(p.Currency)

	// Add interval information only for subscriptions
	if planType == model.PlanTypeSubscription && p.Recurring != nil {
		features["interval"] = string(p.Recurring.Interval)
		features["interval_count"] = p.Recurring.IntervalCount
	}

	// Add product description if available
	if prod.Description != "" {
		features["description"] = prod.Description
	}

	// Extract sort order
	sortOrder := 0
	if order, ok := prod.Metadata["sort_order"]; ok {
		fmt.Sscanf(order, "%d", &sortOrder)
	}

	plan := &model.SubscriptionPlan{
		StripePriceID:   p.ID,
		StripeProductID: prod.ID,
		DisplayName:     prod.Name,
		Type:            planType,
		CreditsPerCycle: creditsPerCycle,
		Features:        features,
		SortOrder:       sortOrder,
		IsActive:        p.Active && prod.Active,
	}

	return s.planRepo.Upsert(ctx, plan)
}
