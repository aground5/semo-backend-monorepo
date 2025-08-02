package repository

import (
	"context"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *entity.Payment) error
	GetByID(ctx context.Context, id string) (*entity.Payment, error)
	GetByUserID(ctx context.Context, userID string) ([]*entity.Payment, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*entity.Payment, error)
	Update(ctx context.Context, payment *entity.Payment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*entity.Payment, error)
	GetRecentByUserID(ctx context.Context, userID string, limit int) ([]*entity.Payment, error)
}
