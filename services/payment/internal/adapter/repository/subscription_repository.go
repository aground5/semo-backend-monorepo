package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	creditRepo          repository.CreditRepository
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *gorm.DB, logger *zap.Logger, customerMappingRepo repository.CustomerMappingRepository, creditRepo repository.CreditRepository) repository.SubscriptionRepository {
	return &subscriptionRepository{
		db:                  db,
		logger:              logger,
		customerMappingRepo: customerMappingRepo,
		creditRepo:          creditRepo,
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

// Cancel cancels a subscription and resets the user's credit balance
func (r *subscriptionRepository) Cancel(ctx context.Context, subscriptionID string) error {
	// Use a database transaction to ensure atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, get the subscription to retrieve user information
		var subscription model.Subscription
		err := tx.Where("stripe_subscription_id = ?", subscriptionID).First(&subscription).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				r.logger.Error("Subscription not found for cancellation",
					zap.String("subscription_id", subscriptionID))
				return fmt.Errorf("subscription not found: %s", subscriptionID)
			}
			r.logger.Error("Failed to retrieve subscription for cancellation",
				zap.String("subscription_id", subscriptionID),
				zap.Error(err))
			return fmt.Errorf("failed to retrieve subscription: %w", err)
		}

		// Update subscription status
		now := time.Now()
		err = tx.Model(&model.Subscription{}).
			Where("stripe_subscription_id = ?", subscriptionID).
			Updates(map[string]interface{}{
				"status":      model.SubscriptionStatusInactive,
				"canceled_at": &now,
			}).Error

		if err != nil {
			r.logger.Error("Failed to update subscription status",
				zap.String("subscription_id", subscriptionID),
				zap.Error(err))
			return fmt.Errorf("failed to update subscription status: %w", err)
		}

		r.logger.Info("Subscription status updated to inactive",
			zap.String("subscription_id", subscriptionID),
			zap.String("universal_id", subscription.UniversalID.String()))

		// Get current balance before resetting
		var currentBalance model.UserCreditBalance
		err = tx.Where("universal_id = ?", subscription.UniversalID).First(&currentBalance).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Error("Failed to retrieve current balance",
				zap.String("universal_id", subscription.UniversalID.String()),
				zap.Error(err))
			return fmt.Errorf("failed to retrieve current balance: %w", err)
		}

		// Only proceed if there's a balance to reset
		if err == nil && currentBalance.CurrentBalance.IsPositive() {
			// Create a transaction record for the balance reset
			balanceResetTransaction := &model.CreditTransaction{
				UniversalID:     subscription.UniversalID,
				SubscriptionID:  &subscription.ID,
				TransactionType: model.TransactionTypeSubscriptionCancellation,
				Amount:          currentBalance.CurrentBalance.Neg(), // Negative amount to zero out balance
				BalanceAfter:    decimal.Zero,
				Description:     fmt.Sprintf("Credit balance reset due to subscription cancellation (Subscription ID: %s)", subscriptionID),
				CreatedAt:       now,
			}

			err = tx.Create(balanceResetTransaction).Error
			if err != nil {
				r.logger.Error("Failed to create balance reset transaction",
					zap.String("universal_id", subscription.UniversalID.String()),
					zap.String("subscription_id", subscriptionID),
					zap.Error(err))
				return fmt.Errorf("failed to create balance reset transaction: %w", err)
			}

			r.logger.Info("Created balance reset transaction",
				zap.String("universal_id", subscription.UniversalID.String()),
				zap.String("amount_reset", currentBalance.CurrentBalance.String()),
				zap.Int64("transaction_id", balanceResetTransaction.ID))

			// Update the user's credit balance to zero
			err = tx.Model(&model.UserCreditBalance{}).
				Where("universal_id = ?", subscription.UniversalID).
				Updates(map[string]interface{}{
					"current_balance":     decimal.Zero,
					"last_transaction_at": now,
				}).Error

			if err != nil {
				r.logger.Error("Failed to reset user credit balance",
					zap.String("universal_id", subscription.UniversalID.String()),
					zap.Error(err))
				return fmt.Errorf("failed to reset user credit balance: %w", err)
			}

			r.logger.Info("User credit balance reset to zero",
				zap.String("universal_id", subscription.UniversalID.String()),
				zap.String("previous_balance", currentBalance.CurrentBalance.String()))
		} else {
			r.logger.Info("No balance to reset for user",
				zap.String("universal_id", subscription.UniversalID.String()))
		}

		r.logger.Info("Subscription canceled successfully",
			zap.String("subscription_id", subscriptionID),
			zap.String("universal_id", subscription.UniversalID.String()))

		return nil
	})
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
		ProductName:       m.ProductName,
		Amount:            m.Amount,
		Currency:          m.Currency,
		Interval:          m.Interval,
		IntervalCount:     m.IntervalCount,
	}

	// If subscription item fields are empty, try to populate from plan (for backward compatibility)
	if e.ProductName == "" && m.Plan != nil {
		e.ProductName = m.Plan.DisplayName
		e.Amount = int64(m.Plan.CreditsPerCycle)
		e.Currency = "KRW"
		e.Interval = "month"
		e.IntervalCount = 1
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

	// Parse the universal ID from the mapping
	universalID, err := uuid.Parse(customerMapping.UniversalID)
	if err != nil {
		r.logger.Error("Failed to parse universal ID from customer mapping",
			zap.String("universal_id", customerMapping.UniversalID),
			zap.Error(err))
		return nil, fmt.Errorf("invalid universal ID in customer mapping: %w", err)
	}

	m := &model.Subscription{
		UniversalID:          universalID,
		StripeCustomerID:     e.CustomerID,
		StripeSubscriptionID: &e.ID,
		PlanID:               e.PlanID,  // PlanID 매핑 추가
		Status:               r.mapEntityStatus(e.Status),
		CurrentPeriodStart:   e.CreatedAt,
		CurrentPeriodEnd:     e.CurrentPeriodEnd,
		ProductName:          e.ProductName,
		Amount:               e.Amount,
		Currency:             e.Currency,
		Interval:             e.Interval,
		IntervalCount:        e.IntervalCount,
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
