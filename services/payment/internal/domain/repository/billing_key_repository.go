package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
)

type BillingKeyRepository interface {
	Create(ctx context.Context, billingKey *model.BillingKey) error
	GetByID(ctx context.Context, id int64) (*model.BillingKey, error)
	GetByCustomerKey(ctx context.Context, customerKey string) (*model.BillingKey, error)
	GetActiveByUniversalID(ctx context.Context, universalID uuid.UUID) ([]*model.BillingKey, error)
	Deactivate(ctx context.Context, id int64) error
	CreateAccessLog(ctx context.Context, log *model.BillingKeyAccessLog) error
}
