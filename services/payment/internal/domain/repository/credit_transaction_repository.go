package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
)

// CreditTransactionRepository defines the interface for credit transaction data operations
type CreditTransactionRepository interface {
	// GetTransactions retrieves credit transactions with filters
	GetTransactions(ctx context.Context, filters dto.TransactionFilters) ([]model.CreditTransaction, error)
	
	// CountTransactions counts the total number of transactions matching the filters
	CountTransactions(ctx context.Context, filters dto.TransactionFilters) (int64, error)
	
	// GetCreditBalance retrieves the current credit balance for a universal ID
	GetCreditBalance(ctx context.Context, universalID uuid.UUID) (*model.UserCreditBalance, error)
}