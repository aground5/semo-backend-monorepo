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
	serviceProvider string
}

// NewCreditTransactionService creates a new credit transaction service
func NewCreditTransactionService(
	transactionRepo repository.CreditTransactionRepository,
	logger *zap.Logger,
	serviceProvider string,
) *CreditTransactionService {
	if serviceProvider == "" {
		logger.Error("CreditTransactionService initialized without service provider")
	}
	return &CreditTransactionService{
		transactionRepo: transactionRepo,
		logger:          logger,
		serviceProvider: serviceProvider,
	}
}

// GetUserTransactionHistory retrieves a user's transaction history with pagination and filters
func (s *CreditTransactionService) GetUserTransactionHistory(
	ctx context.Context,
	universalID uuid.UUID,
	filters dto.TransactionFilters,
) (*dto.TransactionListResponse, error) {
	// Set user ID and defaults
	filters.UserID = universalID
	filters.SetDefaults()

	// Get transactions
	transactions, err := s.transactionRepo.GetTransactions(ctx, filters)
	if err != nil {
		s.logger.Error("failed to get transactions",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	// Get total count for pagination
	totalCount, err := s.transactionRepo.CountTransactions(ctx, filters)
	if err != nil {
		s.logger.Error("failed to count transactions",
			zap.String("universal_id", universalID.String()),
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

// GetCreditBalance retrieves the current credit balance for a universal ID
func (s *CreditTransactionService) GetCreditBalance(ctx context.Context, universalID uuid.UUID) (string, error) {
	balance, err := s.transactionRepo.GetCreditBalance(ctx, universalID, s.serviceProvider)
	if err != nil {
		s.logger.Error("failed to get universal ID credit balance",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return "", fmt.Errorf("failed to get credit balance: %w", err)
	}

	if balance == nil {
		return "0.00", nil
	}

	return balance.CurrentBalance.String(), nil
}
