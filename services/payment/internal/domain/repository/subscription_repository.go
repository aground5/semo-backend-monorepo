package repository

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
)

type SubscriptionRepository interface {
	GetByCustomerID(ctx context.Context, customerID string) (*entity.Subscription, error)
	GetByID(ctx context.Context, subscriptionID string) (*entity.Subscription, error)
	Save(ctx context.Context, subscription *entity.Subscription) error
	Update(ctx context.Context, subscription *entity.Subscription) error
	Cancel(ctx context.Context, subscriptionID string) error
	ListByStatus(ctx context.Context, status string) ([]*entity.Subscription, error)
}