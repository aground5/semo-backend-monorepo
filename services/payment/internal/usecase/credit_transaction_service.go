package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
)

// CreditTransactionService handles credit transaction business logic
type CreditTransactionService struct {
	transactionRepo repository.CreditTransactionRepository
	logger          *zap.Logger
}

// NewCreditTransactionService creates a new credit transaction service
func NewCreditTransactionService(
	transactionRepo repository.CreditTransactionRepository,
	logger *zap.Logger,
) *CreditTransactionService {
	return &CreditTransactionService{
		transactionRepo: transactionRepo,
		logger:          logger,
	}
}

// GetUserTransactionHistory retrieves a user's transaction history with pagination and filters
func (s *CreditTransactionService) GetUserTransactionHistory(
	ctx context.Context,
	userID uuid.UUID,
	filters dto.TransactionFilters,
) (*dto.TransactionListResponse, error) {
	// Set user ID and defaults
	filters.UserID = userID
	filters.SetDefaults()

	// Get transactions
	transactions, err := s.transactionRepo.GetUserTransactions(ctx, filters)
	if err != nil {
		s.logger.Error("failed to get user transactions",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	// Get total count for pagination
	totalCount, err := s.transactionRepo.CountUserTransactions(ctx, filters)
	if err != nil {
		s.logger.Error("failed to count user transactions",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Transform to DTOs
	transactionDTOs := make([]dto.CreditTransactionDTO, len(transactions))
	for i, tx := range transactions {
		// Format amount with sign
		amountStr := tx.Amount.String()
		if tx.TransactionType == "credit_usage" && !strings.HasPrefix(amountStr, "-") {
			amountStr = "-" + amountStr
		}

		// Truncate description to 50 characters
		description := tx.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		transactionDTOs[i] = dto.CreditTransactionDTO{
			TransactionType: string(tx.TransactionType),
			Amount:          amountStr,
			BalanceAfter:    tx.BalanceAfter.String(),
			Description:     description,
			CreatedAt:       tx.CreatedAt,
		}
	}

	// Calculate pagination info
	hasMore := int64(filters.Offset+filters.Limit) < totalCount

	response := &dto.TransactionListResponse{
		Transactions: transactionDTOs,
		Pagination: dto.PaginationInfo{
			Total:   totalCount,
			Limit:   filters.Limit,
			Offset:  filters.Offset,
			HasMore: hasMore,
		},
	}

	return response, nil
}

// GetUserCreditBalance retrieves the current credit balance for a user
func (s *CreditTransactionService) GetUserCreditBalance(ctx context.Context, userID uuid.UUID) (string, error) {
	balance, err := s.transactionRepo.GetUserCreditBalance(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user credit balance",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return "", fmt.Errorf("failed to get credit balance: %w", err)
	}

	if balance == nil {
		return "0.00", nil
	}

	return balance.CurrentBalance.String(), nil
}