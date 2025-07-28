package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PlanRepository handles subscription plan storage
type PlanRepository interface {
	GetAll(ctx context.Context) ([]*model.SubscriptionPlan, error)
	GetByType(ctx context.Context, planType string) ([]*model.SubscriptionPlan, error)
	GetByPriceID(ctx context.Context, priceID string) (*model.SubscriptionPlan, error)
	GetByProductID(ctx context.Context, productID string) ([]*model.SubscriptionPlan, error)
	Create(ctx context.Context, plan *model.SubscriptionPlan) error
	Update(ctx context.Context, plan *model.SubscriptionPlan) error
	Delete(ctx context.Context, priceID string) error
	Upsert(ctx context.Context, plan *model.SubscriptionPlan) error
}

type planRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPlanRepository creates a new plan repository
func NewPlanRepository(db *gorm.DB, logger *zap.Logger) PlanRepository {
	return &planRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll retrieves all active subscription plans
func (r *planRepository) GetAll(ctx context.Context) ([]*model.SubscriptionPlan, error) {
	var plans []*model.SubscriptionPlan

	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC, display_name ASC").
		Find(&plans).Error

	if err != nil {
		r.logger.Error("Failed to get all plans", zap.Error(err))
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}

	return plans, nil
}

// GetByType retrieves all active plans of a specific type
func (r *planRepository) GetByType(ctx context.Context, planType string) ([]*model.SubscriptionPlan, error) {
	var plans []*model.SubscriptionPlan

	err := r.db.WithContext(ctx).
		Where("type = ? AND is_active = ?", planType, true).
		Order("sort_order ASC, display_name ASC").
		Find(&plans).Error

	if err != nil {
		r.logger.Error("Failed to get plans by type",
			zap.String("type", planType),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get plans by type: %w", err)
	}

	return plans, nil
}

// GetByPriceID retrieves a plan by Stripe price ID
func (r *planRepository) GetByPriceID(ctx context.Context, priceID string) (*model.SubscriptionPlan, error) {
	var plan model.SubscriptionPlan

	err := r.db.WithContext(ctx).
		Where("stripe_price_id = ?", priceID).
		First(&plan).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get plan by price ID",
			zap.String("price_id", priceID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	return &plan, nil
}

// GetByProductID retrieves all plans for a Stripe product
func (r *planRepository) GetByProductID(ctx context.Context, productID string) ([]*model.SubscriptionPlan, error) {
	var plans []*model.SubscriptionPlan

	err := r.db.WithContext(ctx).
		Where("stripe_product_id = ?", productID).
		Find(&plans).Error

	if err != nil {
		r.logger.Error("Failed to get plans by product ID",
			zap.String("product_id", productID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}

	return plans, nil
}

// Create creates a new subscription plan
func (r *planRepository) Create(ctx context.Context, plan *model.SubscriptionPlan) error {
	err := r.db.WithContext(ctx).Create(plan).Error
	if err != nil {
		r.logger.Error("Failed to create plan",
			zap.String("price_id", plan.StripePriceID),
			zap.Error(err))
		return fmt.Errorf("failed to create plan: %w", err)
	}

	return nil
}

// Update updates an existing subscription plan
func (r *planRepository) Update(ctx context.Context, plan *model.SubscriptionPlan) error {
	err := r.db.WithContext(ctx).
		Model(&model.SubscriptionPlan{}).
		Where("stripe_price_id = ?", plan.StripePriceID).
		Updates(plan).Error

	if err != nil {
		r.logger.Error("Failed to update plan",
			zap.String("price_id", plan.StripePriceID),
			zap.Error(err))
		return fmt.Errorf("failed to update plan: %w", err)
	}

	return nil
}

// Delete soft deletes a subscription plan
func (r *planRepository) Delete(ctx context.Context, priceID string) error {
	err := r.db.WithContext(ctx).
		Model(&model.SubscriptionPlan{}).
		Where("stripe_price_id = ?", priceID).
		Update("is_active", false).Error

	if err != nil {
		r.logger.Error("Failed to delete plan",
			zap.String("price_id", priceID),
			zap.Error(err))
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	return nil
}

// Upsert creates or updates a subscription plan
func (r *planRepository) Upsert(ctx context.Context, plan *model.SubscriptionPlan) error {
	// Check if plan exists
	existing, err := r.GetByPriceID(ctx, plan.StripePriceID)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing plan
		plan.ID = existing.ID
		return r.Update(ctx, plan)
	}

	// Create new plan
	return r.Create(ctx, plan)
}
