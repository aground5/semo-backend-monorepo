package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
)

// CreditRepository defines the interface for credit-related operations
type CreditRepository interface {
	// GetBalance retrieves the current credit balance for a user
	GetBalance(ctx context.Context, userID uuid.UUID) (*model.UserCreditBalance, error)

	// AllocateCredits adds credits to a user's balance atomically
	// Returns the new balance and the created transaction
	AllocateCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, description string, referenceID string) (*model.UserCreditBalance, *model.CreditTransaction, error)

	// UseCredits deducts credits from a user's balance atomically
	// Returns the new balance and the created transaction
	UseCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, description string, featureName string) (*model.UserCreditBalance, *model.CreditTransaction, error)

	// GetTransactionByReference retrieves a transaction by its reference ID (for idempotency)
	GetTransactionByReference(ctx context.Context, referenceID string) (*model.CreditTransaction, error)

	// GetTransactionHistory retrieves transaction history for a user
	GetTransactionHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.CreditTransaction, error)
}
