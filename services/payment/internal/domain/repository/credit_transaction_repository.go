package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
)

// CreditTransactionRepository defines the interface for credit transaction data operations
type CreditTransactionRepository interface {
	// GetUserTransactions retrieves a user's credit transactions with filters
	GetUserTransactions(ctx context.Context, filters dto.TransactionFilters) ([]model.CreditTransaction, error)
	
	// CountUserTransactions counts the total number of transactions matching the filters
	CountUserTransactions(ctx context.Context, filters dto.TransactionFilters) (int64, error)
	
	// GetUserCreditBalance retrieves the current credit balance for a user
	GetUserCreditBalance(ctx context.Context, userID uuid.UUID) (*model.UserCreditBalance, error)
}