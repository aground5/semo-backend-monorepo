package repository

import (
	"context"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
)

type CustomerMappingRepository interface {
	Create(ctx context.Context, mapping *entity.CustomerMapping) error
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*entity.CustomerMapping, error)
	GetByUserID(ctx context.Context, userID string) (*entity.CustomerMapping, error)
	Update(ctx context.Context, mapping *entity.CustomerMapping) error
}
