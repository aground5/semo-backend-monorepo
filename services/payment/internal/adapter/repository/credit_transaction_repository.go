package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
)

// creditTransactionRepository implements the CreditTransactionRepository interface
type creditTransactionRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewCreditTransactionRepository creates a new credit transaction repository
func NewCreditTransactionRepository(db *gorm.DB, logger *zap.Logger) repository.CreditTransactionRepository {
	return &creditTransactionRepository{
		db:     db,
		logger: logger,
	}
}

// GetUserTransactions retrieves a user's credit transactions with filters
func (r *creditTransactionRepository) GetUserTransactions(ctx context.Context, filters dto.TransactionFilters) ([]model.CreditTransaction, error) {
	var transactions []model.CreditTransaction
	
	query := r.db.WithContext(ctx).
		Where("user_id = ?", filters.UserID).
		Order("created_at DESC")

	// Apply date filters
	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", *filters.EndDate)
	}

	// Apply transaction type filter
	if filters.TransactionType != nil && *filters.TransactionType != "" {
		query = query.Where("transaction_type = ?", *filters.TransactionType)
	}

	// Apply pagination
	query = query.Limit(filters.Limit).Offset(filters.Offset)

	if err := query.Find(&transactions).Error; err != nil {
		r.logger.Error("failed to get user transactions",
			zap.String("user_id", filters.UserID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get user transactions: %w", err)
	}

	return transactions, nil
}

// CountUserTransactions counts the total number of transactions matching the filters
func (r *creditTransactionRepository) CountUserTransactions(ctx context.Context, filters dto.TransactionFilters) (int64, error) {
	var count int64
	
	query := r.db.WithContext(ctx).
		Model(&model.CreditTransaction{}).
		Where("user_id = ?", filters.UserID)

	// Apply date filters
	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", *filters.EndDate)
	}

	// Apply transaction type filter
	if filters.TransactionType != nil && *filters.TransactionType != "" {
		query = query.Where("transaction_type = ?", *filters.TransactionType)
	}

	if err := query.Count(&count).Error; err != nil {
		r.logger.Error("failed to count user transactions",
			zap.String("user_id", filters.UserID.String()),
			zap.Error(err))
		return 0, fmt.Errorf("failed to count user transactions: %w", err)
	}

	return count, nil
}

// GetUserCreditBalance retrieves the current credit balance for a user
func (r *creditTransactionRepository) GetUserCreditBalance(ctx context.Context, userID uuid.UUID) (*model.UserCreditBalance, error) {
	var balance model.UserCreditBalance
	
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&balance).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get user credit balance",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get user credit balance: %w", err)
	}

	return &balance, nil
}