package repository

import (
	"context"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
)

type CustomerMappingRepository interface {
	Create(ctx context.Context, mapping *entity.CustomerMapping) error
	GetByProviderCustomerID(ctx context.Context, provider string, providerCustomerID string) (*entity.CustomerMapping, error)
	GetByProviderAndUniversalID(ctx context.Context, provider string, universalID string) (*entity.CustomerMapping, error)
	Update(ctx context.Context, mapping *entity.CustomerMapping) error
}
