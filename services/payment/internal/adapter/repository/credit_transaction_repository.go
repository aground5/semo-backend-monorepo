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

// GetTransactions retrieves credit transactions with filters
func (r *creditTransactionRepository) GetTransactions(ctx context.Context, filters dto.TransactionFilters) ([]model.CreditTransaction, error) {
	var transactions []model.CreditTransaction

	query := r.db.WithContext(ctx).
		Where("universal_id = ?", filters.UserID).
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
		r.logger.Error("failed to get transactions",
			zap.String("universal_id", filters.UserID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, nil
}

// CountTransactions counts the total number of transactions matching the filters
func (r *creditTransactionRepository) CountTransactions(ctx context.Context, filters dto.TransactionFilters) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).
		Model(&model.CreditTransaction{}).
		Where("universal_id = ?", filters.UserID)

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
		r.logger.Error("failed to count transactions",
			zap.String("universal_id", filters.UserID.String()),
			zap.Error(err))
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

// GetCreditBalance retrieves the current credit balance for a universal ID
func (r *creditTransactionRepository) GetCreditBalance(ctx context.Context, universalID uuid.UUID, serviceProvider string) (*model.UserCreditBalance, error) {
	var balance model.UserCreditBalance

	err := r.db.WithContext(ctx).
		Where("universal_id = ? AND service_provider = ?", universalID, serviceProvider).
		First(&balance).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get universal ID credit balance",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get universal ID credit balance: %w", err)
	}

	balance.ServiceProvider = serviceProvider
	return &balance, nil
}
