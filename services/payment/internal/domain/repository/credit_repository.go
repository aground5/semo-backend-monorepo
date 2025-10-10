package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
)

// CreditRepository defines the interface for credit-related operations
type CreditRepository interface {
	// GetBalance retrieves the current credit balance for a universal ID
	GetBalance(ctx context.Context, universalID uuid.UUID, serviceProvider string) (*model.UserCreditBalance, error)

	// AllocateCredits adds credits to a universal ID's balance atomically
	// Returns the new balance and the created transaction
	AllocateCredits(ctx context.Context, universalID uuid.UUID, serviceProvider string, amount decimal.Decimal, description string, referenceID string) (*model.UserCreditBalance, *model.CreditTransaction, error)

	// UseCredits deducts credits from a universal ID's balance atomically
	// Returns the new balance and the created transaction
	UseCredits(ctx context.Context, universalID uuid.UUID, serviceProvider string, amount decimal.Decimal, description string, featureName string) (*model.UserCreditBalance, *model.CreditTransaction, error)

	// GetTransactionByReference retrieves a transaction by its reference ID (for idempotency)
	GetTransactionByReference(ctx context.Context, referenceID string) (*model.CreditTransaction, error)

	// GetTransactionHistory retrieves transaction history for a universal ID
	GetTransactionHistory(ctx context.Context, universalID uuid.UUID, limit, offset int) ([]*model.CreditTransaction, error)
}
