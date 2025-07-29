package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type subscriptionRepository struct {
	db                  *gorm.DB
	logger              *zap.Logger
	customerMappingRepo repository.CustomerMappingRepository
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *gorm.DB, logger *zap.Logger, customerMappingRepo repository.CustomerMappingRepository) repository.SubscriptionRepository {
	return &subscriptionRepository{
		db:                  db,
		logger:              logger,
		customerMappingRepo: customerMappingRepo,
	}
}

// GetByCustomerID retrieves subscription by Stripe customer ID
func (r *subscriptionRepository) GetByCustomerID(ctx context.Context, customerID string) (*entity.Subscription, error) {
	var sub model.Subscription

	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("stripe_customer_id = ? AND status = ?", customerID, model.SubscriptionStatusActive).
		First(&sub).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get subscription by customer ID",
			zap.String("customer_id", customerID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return r.modelToEntity(&sub), nil
}

// GetByID retrieves subscription by Stripe subscription ID
func (r *subscriptionRepository) GetByID(ctx context.Context, subscriptionID string) (*entity.Subscription, error) {
	var sub model.Subscription

	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("stripe_subscription_id = ?", subscriptionID).
		First(&sub).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get subscription by ID",
			zap.String("subscription_id", subscriptionID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return r.modelToEntity(&sub), nil
}

// Save creates a new subscription
func (r *subscriptionRepository) Save(ctx context.Context, subscription *entity.Subscription) error {
	sub, err := r.entityToModel(ctx, subscription)
	if err != nil {
		return err
	}

	err = r.db.WithContext(ctx).Create(sub).Error
	if err != nil {
		r.logger.Error("Failed to save subscription",
			zap.String("customer_id", subscription.CustomerID),
			zap.Error(err))
		return fmt.Errorf("failed to save subscription: %w", err)
	}

	return nil
}

// Update updates an existing subscription
func (r *subscriptionRepository) Update(ctx context.Context, subscription *entity.Subscription) error {
	// First check if subscription exists
	var existing model.Subscription
	err := r.db.WithContext(ctx).
		Where("stripe_subscription_id = ?", subscription.ID).
		First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("subscription not found: %s", subscription.ID)
		}
		return fmt.Errorf("failed to check subscription: %w", err)
	}

	// Update fields
	updates := map[string]interface{}{
		"status":             r.mapEntityStatus(subscription.Status),
		"current_period_end": subscription.CurrentPeriodEnd,
		"canceled_at":        nil,
	}

	if subscription.CancelAtPeriodEnd {
		now := time.Now()
		updates["canceled_at"] = &now
	}

	err = r.db.WithContext(ctx).
		Model(&model.Subscription{}).
		Where("stripe_subscription_id = ?", subscription.ID).
		Updates(updates).Error

	if err != nil {
		r.logger.Error("Failed to update subscription",
			zap.String("subscription_id", subscription.ID),
			zap.Error(err))
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// Cancel cancels a subscription
func (r *subscriptionRepository) Cancel(ctx context.Context, subscriptionID string) error {
	now := time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.Subscription{}).
		Where("stripe_subscription_id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"status":      model.SubscriptionStatusInactive,
			"canceled_at": &now,
		}).Error

	if err != nil {
		r.logger.Error("Failed to cancel subscription",
			zap.String("subscription_id", subscriptionID),
			zap.Error(err))
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	return nil
}

// ListByStatus lists subscriptions by status
func (r *subscriptionRepository) ListByStatus(ctx context.Context, status string) ([]*entity.Subscription, error) {
	var subs []model.Subscription

	query := r.db.WithContext(ctx).Preload("Plan")
	if status != "" {
		query = query.Where("status = ?", r.mapEntityStatus(status))
	}

	err := query.Find(&subs).Error
	if err != nil {
		r.logger.Error("Failed to list subscriptions by status",
			zap.String("status", status),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	entities := make([]*entity.Subscription, len(subs))
	for i, sub := range subs {
		entities[i] = r.modelToEntity(&sub)
	}

	return entities, nil
}

// modelToEntity converts database model to domain entity
func (r *subscriptionRepository) modelToEntity(m *model.Subscription) *entity.Subscription {
	if m == nil {
		return nil
	}

	e := &entity.Subscription{
		ID:                *m.StripeSubscriptionID,
		CustomerID:        m.StripeCustomerID,
		Status:            string(m.Status),
		CurrentPeriodEnd:  m.CurrentPeriodEnd,
		CancelAtPeriodEnd: m.CanceledAt != nil,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		Items:             []entity.SubscriptionItem{},
	}

	// Add subscription items from plan
	if m.Plan != nil {
		e.Items = append(e.Items, entity.SubscriptionItem{
			ProductName:   m.Plan.DisplayName,
			Amount:        int64(m.Plan.CreditsPerCycle),
			Currency:      "KRW",
			Interval:      "month",
			IntervalCount: 1,
		})
	}

	return e
}

// entityToModel converts domain entity to database model
func (r *subscriptionRepository) entityToModel(ctx context.Context, e *entity.Subscription) (*model.Subscription, error) {
	if e == nil {
		return nil, nil
	}

	// Look up user ID from customer mapping
	customerMapping, err := r.customerMappingRepo.GetByStripeCustomerID(ctx, e.CustomerID)
	if err != nil {
		r.logger.Error("Failed to get customer mapping",
			zap.String("stripe_customer_id", e.CustomerID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get customer mapping: %w", err)
	}
	
	if customerMapping == nil {
		r.logger.Error("Customer mapping not found",
			zap.String("stripe_customer_id", e.CustomerID))
		return nil, fmt.Errorf("customer mapping not found for stripe customer ID: %s", e.CustomerID)
	}

	// Parse the user ID from the mapping
	userID, err := uuid.Parse(customerMapping.UserID)
	if err != nil {
		r.logger.Error("Failed to parse user ID from customer mapping",
			zap.String("user_id", customerMapping.UserID),
			zap.Error(err))
		return nil, fmt.Errorf("invalid user ID in customer mapping: %w", err)
	}

	m := &model.Subscription{
		UserID:               userID,
		StripeCustomerID:     e.CustomerID,
		StripeSubscriptionID: &e.ID,
		Status:               r.mapEntityStatus(e.Status),
		CurrentPeriodStart:   e.CreatedAt, // Approximate
		CurrentPeriodEnd:     e.CurrentPeriodEnd,
	}

	if e.CancelAtPeriodEnd {
		now := time.Now()
		m.CanceledAt = &now
	}

	return m, nil
}

// mapEntityStatus maps entity status to model status
func (r *subscriptionRepository) mapEntityStatus(status string) model.SubscriptionStatus {
	switch status {
	case "active", "trialing":
		return model.SubscriptionStatusActive
	default:
		return model.SubscriptionStatusInactive
	}
}
