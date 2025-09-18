package repository

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *entity.Payment) error
	GetByID(ctx context.Context, id string) (*entity.Payment, error)
	GetByUniversalID(ctx context.Context, universalID string, page, limit int) ([]*entity.Payment, int64, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*entity.Payment, error)
	Update(ctx context.Context, payment *entity.Payment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*entity.Payment, error)

	// One-time payment methods
	CreateOneTimePayment(ctx context.Context, payment *entity.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error)
	UpdatePaymentAfterConfirm(ctx context.Context, orderID string, updates map[string]interface{}) error
}
