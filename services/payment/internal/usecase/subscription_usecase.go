package usecase

import (
	"context"
	"errors"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

type SubscriptionUsecase struct {
	subscriptionRepo repository.SubscriptionRepository
	logger          *zap.Logger
}

func NewSubscriptionUsecase(subscriptionRepo repository.SubscriptionRepository, logger *zap.Logger) *SubscriptionUsecase {
	return &SubscriptionUsecase{
		subscriptionRepo: subscriptionRepo,
		logger:          logger,
	}
}

func (u *SubscriptionUsecase) GetCurrentSubscription(ctx context.Context, customerID string) (*entity.Subscription, error) {
	if customerID == "" {
		return nil, errors.New("customer ID is required")
	}

	subscription, err := u.subscriptionRepo.GetByCustomerID(ctx, customerID)
	if err != nil {
		u.logger.Error("Failed to get subscription", zap.Error(err))
		return nil, err
	}

	return subscription, nil
}

func (u *SubscriptionUsecase) CancelSubscription(ctx context.Context, subscriptionID string) error {
	if subscriptionID == "" {
		return errors.New("subscription ID is required")
	}

	err := u.subscriptionRepo.Cancel(ctx, subscriptionID)
	if err != nil {
		u.logger.Error("Failed to cancel subscription", zap.Error(err))
		return err
	}

	return nil
}

func (u *SubscriptionUsecase) SaveSubscription(ctx context.Context, subscription *entity.Subscription) error {
	if subscription == nil {
		return errors.New("subscription is required")
	}

	err := u.subscriptionRepo.Save(ctx, subscription)
	if err != nil {
		u.logger.Error("Failed to save subscription", zap.Error(err))
		return err
	}

	return nil
}