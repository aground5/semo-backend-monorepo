package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// creditRepository implements the CreditRepository interface
type creditRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewCreditRepository creates a new credit repository instance
func NewCreditRepository(db *gorm.DB, logger *zap.Logger) domainRepo.CreditRepository {
	return &creditRepository{
		db:     db,
		logger: logger,
	}
}

// GetBalance retrieves the current credit balance for a universal ID
func (r *creditRepository) GetBalance(ctx context.Context, universalID uuid.UUID) (*model.UserCreditBalance, error) {
	var balance model.UserCreditBalance

	err := r.db.WithContext(ctx).
		Where("universal_id = ?", universalID).
		First(&balance).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return zero balance if not found
			return &model.UserCreditBalance{
				UniversalID:    universalID,
				CurrentBalance: decimal.Zero,
			}, nil
		}
		r.logger.Error("Failed to get credit balance",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get credit balance: %w", err)
	}

	return &balance, nil
}

// AllocateCredits adds credits to a universal ID's balance atomically
func (r *creditRepository) AllocateCredits(ctx context.Context, universalID uuid.UUID, amount decimal.Decimal, description string, referenceID string) (*model.UserCreditBalance, *model.CreditTransaction, error) {
	var balance *model.UserCreditBalance
	var transaction *model.CreditTransaction

	// Use a database transaction for atomicity
	r.logger.Info("Starting database transaction for credit allocation",
		zap.String("universal_id", universalID.String()),
		zap.String("amount", amount.String()),
		zap.String("reference_id", referenceID))

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check for existing transaction with same reference ID (idempotency)
		if referenceID != "" {
			var existingTx model.CreditTransaction
			err := tx.Where("reference_id = ?", referenceID).First(&existingTx).Error
			if err == nil {
				// Transaction already exists, return existing data
				transaction = &existingTx

				// Get current balance
				var currentBalance model.UserCreditBalance
				if err := tx.Where("universal_id = ?", universalID).First(&currentBalance).Error; err == nil {
					balance = &currentBalance
				}

				r.logger.Info("Credit allocation already processed (idempotency)",
					zap.String("reference_id", referenceID),
					zap.String("universal_id", universalID.String()))
				return nil
			}
		}

		// Lock the user's balance row for update (or create if doesn't exist)
		var currentBalance model.UserCreditBalance
		r.logger.Info("Attempting to lock/create user balance row",
			zap.String("universal_id", universalID.String()),
			zap.String("transaction_type", "credit_allocation"))

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("universal_id = ?", universalID).
			FirstOrCreate(&currentBalance, model.UserCreditBalance{
				UniversalID:    universalID,
				CurrentBalance: decimal.Zero,
			}).Error

		if err != nil {
			r.logger.Error("Failed to lock/create balance row",
				zap.String("universal_id", universalID.String()),
				zap.Error(err))
			return fmt.Errorf("failed to lock balance: %w", err)
		}

		r.logger.Info("Successfully locked/created balance row",
			zap.String("universal_id", universalID.String()),
			zap.String("current_balance", currentBalance.CurrentBalance.String()),
			zap.Bool("was_created", currentBalance.LastTransactionAt.IsZero()))

		// Calculate new balance
		newBalance := currentBalance.CurrentBalance.Add(amount)

		// Create transaction record
		transaction = &model.CreditTransaction{
			UniversalID:     universalID,
			TransactionType: model.TransactionTypeCreditAllocation,
			Amount:          amount,
			BalanceAfter:    newBalance,
			Description:     description,
			ReferenceID:     &referenceID,
		}

		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Update balance
		currentBalance.CurrentBalance = newBalance
		currentBalance.LastTransactionAt = transaction.CreatedAt

		r.logger.Info("Updating user_credit_balances table",
			zap.String("universal_id", universalID.String()),
			zap.String("new_balance", newBalance.String()),
			zap.String("amount", amount.String()),
			zap.Time("last_transaction_at", currentBalance.LastTransactionAt))

		if err := tx.Save(&currentBalance).Error; err != nil {
			r.logger.Error("Failed to update user_credit_balances table",
				zap.String("universal_id", universalID.String()),
				zap.String("new_balance", newBalance.String()),
				zap.Error(err))
			return fmt.Errorf("failed to update balance: %w", err)
		}

		r.logger.Info("Successfully updated user_credit_balances table",
			zap.String("universal_id", universalID.String()),
			zap.String("new_balance", newBalance.String()))

		balance = &currentBalance
		return nil
	})

	if err != nil {
		r.logger.Error("Failed to allocate credits",
			zap.String("universal_id", universalID.String()),
			zap.String("amount", amount.String()),
			zap.String("reference_id", referenceID),
			zap.Error(err))
		return nil, nil, fmt.Errorf("failed to allocate credits: %w", err)
	}

	r.logger.Info("Credits allocated successfully",
		zap.String("universal_id", universalID.String()),
		zap.String("amount", amount.String()),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("reference_id", referenceID))

	return balance, transaction, nil
}

// UseCredits deducts credits from a universal ID's balance atomically
func (r *creditRepository) UseCredits(ctx context.Context, universalID uuid.UUID, amount decimal.Decimal, description string, featureName string) (*model.UserCreditBalance, *model.CreditTransaction, error) {
	var balance *model.UserCreditBalance
	var transaction *model.CreditTransaction

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the user's balance row for update
		var currentBalance model.UserCreditBalance
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("universal_id = ?", universalID).
			First(&currentBalance).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("no credit balance found for user")
			}
			return fmt.Errorf("failed to lock balance: %w", err)
		}

		// Check if sufficient balance
		if currentBalance.CurrentBalance.LessThan(amount) {
			return fmt.Errorf("insufficient credit balance: have %s, need %s",
				currentBalance.CurrentBalance.String(), amount.String())
		}

		// Calculate new balance
		newBalance := currentBalance.CurrentBalance.Sub(amount)

		// Create transaction record
		transaction = &model.CreditTransaction{
			UniversalID:     universalID,
			TransactionType: model.TransactionTypeCreditUsage,
			Amount:          amount.Neg(), // Negative for usage
			BalanceAfter:    newBalance,
			Description:     description,
			FeatureName:     &featureName,
		}

		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Update balance
		currentBalance.CurrentBalance = newBalance
		currentBalance.LastTransactionAt = transaction.CreatedAt

		r.logger.Info("Updating user_credit_balances table",
			zap.String("universal_id", universalID.String()),
			zap.String("new_balance", newBalance.String()),
			zap.String("amount", amount.String()),
			zap.Time("last_transaction_at", currentBalance.LastTransactionAt))

		if err := tx.Save(&currentBalance).Error; err != nil {
			r.logger.Error("Failed to update user_credit_balances table",
				zap.String("universal_id", universalID.String()),
				zap.String("new_balance", newBalance.String()),
				zap.Error(err))
			return fmt.Errorf("failed to update balance: %w", err)
		}

		r.logger.Info("Successfully updated user_credit_balances table",
			zap.String("universal_id", universalID.String()),
			zap.String("new_balance", newBalance.String()))

		balance = &currentBalance
		return nil
	})

	if err != nil {
		r.logger.Error("Failed to use credits",
			zap.String("universal_id", universalID.String()),
			zap.String("amount", amount.String()),
			zap.String("feature", featureName),
			zap.Error(err))
		return nil, nil, fmt.Errorf("failed to use credits: %w", err)
	}

	r.logger.Info("Credits used successfully",
		zap.String("universal_id", universalID.String()),
		zap.String("amount", amount.String()),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("feature", featureName))

	return balance, transaction, nil
}

// GetTransactionByReference retrieves a transaction by its reference ID
func (r *creditRepository) GetTransactionByReference(ctx context.Context, referenceID string) (*model.CreditTransaction, error) {
	var transaction model.CreditTransaction

	err := r.db.WithContext(ctx).
		Where("reference_id = ?", referenceID).
		First(&transaction).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get transaction by reference",
			zap.String("reference_id", referenceID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

// GetTransactionHistory retrieves transaction history for a universal ID
func (r *creditRepository) GetTransactionHistory(ctx context.Context, universalID uuid.UUID, limit, offset int) ([]*model.CreditTransaction, error) {
	var transactions []*model.CreditTransaction

	query := r.db.WithContext(ctx).
		Where("universal_id = ?", universalID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&transactions).Error
	if err != nil {
		r.logger.Error("Failed to get transaction history",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	return transactions, nil
}
