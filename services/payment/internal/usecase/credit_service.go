package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/repository"
	customErr "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

// CreditService handles credit-related business logic
type CreditService struct {
	creditRepo       domainRepo.CreditRepository
	subscriptionRepo domainRepo.SubscriptionRepository
	planRepo         repository.PlanRepository
	logger           *zap.Logger
}

// NewCreditService creates a new credit service instance
func NewCreditService(
	creditRepo domainRepo.CreditRepository,
	subscriptionRepo domainRepo.SubscriptionRepository,
	planRepo repository.PlanRepository,
	logger *zap.Logger,
) *CreditService {
	return &CreditService{
		creditRepo:       creditRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

// AllocateCreditsForPayment allocates credits based on a successful payment
func (s *CreditService) AllocateCreditsForPayment(ctx context.Context, universalID uuid.UUID, invoiceID string, subscriptionID string, stripePriceID string) error {
	s.logger.Info("=== AllocateCreditsForPayment START ===",
		zap.String("universal_id", universalID.String()),
		zap.String("invoice_id", invoiceID),
		zap.String("subscription_id", subscriptionID),
		zap.String("price_id", stripePriceID))

	// First check if credits were already allocated for this invoice (idempotency)
	s.logger.Info("Checking for existing transaction", zap.String("invoice_id", invoiceID))
	existingTx, err := s.creditRepo.GetTransactionByReference(ctx, invoiceID)
	if err != nil {
		s.logger.Error("Failed to check existing transaction",
			zap.String("invoice_id", invoiceID),
			zap.Error(err))
		return fmt.Errorf("failed to check existing transaction: %w", err)
	}

	if existingTx != nil {
		s.logger.Info("Credits already allocated for invoice (idempotency check passed)",
			zap.String("invoice_id", invoiceID),
			zap.String("universal_id", universalID.String()),
			zap.String("existing_tx_id", fmt.Sprintf("%d", existingTx.ID)))
		return nil
	} else {
		s.logger.Info("No existing transaction found, proceeding with credit allocation")
	}

	// Get the subscription plan to determine credits
	s.logger.Info("Looking up subscription plan by price ID", zap.String("price_id", stripePriceID))
	plan, err := s.planRepo.GetByPriceID(ctx, stripePriceID)
	if err != nil {
		s.logger.Error("Failed to get subscription plan from database",
			zap.String("price_id", stripePriceID),
			zap.Error(err))
		return fmt.Errorf("failed to get subscription plan: %w", err)
	}

	if plan == nil {
		s.logger.Error("No subscription plan found for price ID",
			zap.String("price_id", stripePriceID))
		return fmt.Errorf("subscription plan not found for price: %s", stripePriceID)
	} else {
		s.logger.Info("Found subscription plan",
			zap.String("plan_name", plan.DisplayName),
			zap.Int("credits_per_cycle", plan.CreditsPerCycle),
			zap.String("stripe_price_id", plan.StripePriceID))
	}

	// Allocate credits
	amount := decimal.NewFromInt(int64(plan.CreditsPerCycle))
	description := fmt.Sprintf("Credit allocation for %s subscription", plan.DisplayName)

	s.logger.Info("CALLING creditRepo.AllocateCredits",
		zap.String("universal_id", universalID.String()),
		zap.String("amount", amount.String()),
		zap.String("description", description),
		zap.String("reference_id", invoiceID))

	balance, transaction, err := s.creditRepo.AllocateCredits(ctx, universalID, amount, description, invoiceID)
	if err != nil {
		s.logger.Error("CREDIT ALLOCATION FAILED IN REPOSITORY",
			zap.String("universal_id", universalID.String()),
			zap.String("invoice_id", invoiceID),
			zap.String("amount", amount.String()),
			zap.Error(err))
		return fmt.Errorf("failed to allocate credits: %w", err)
	}

	s.logger.Info("CREDIT ALLOCATION SUCCESSFUL",
		zap.String("universal_id", universalID.String()),
		zap.String("invoice_id", invoiceID),
		zap.String("subscription_id", subscriptionID),
		zap.Int("credits", plan.CreditsPerCycle),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("transaction_id", fmt.Sprintf("%d", transaction.ID)))

	s.logger.Info("=== AllocateCreditsForPayment END ===")
	return nil
}

// AllocateCreditsWithMetadata allocates credits based on product metadata
func (s *CreditService) AllocateCreditsWithMetadata(ctx context.Context, universalID uuid.UUID, invoiceID string, creditsPerCycle int, productName string) error {
	s.logger.Info("=== AllocateCreditsWithMetadata START ===",
		zap.String("universal_id", universalID.String()),
		zap.String("invoice_id", invoiceID),
		zap.Int("credits_per_cycle", creditsPerCycle),
		zap.String("product_name", productName))

	// First check if credits were already allocated for this invoice (idempotency)
	s.logger.Info("Checking for existing transaction", zap.String("invoice_id", invoiceID))
	existingTx, err := s.creditRepo.GetTransactionByReference(ctx, invoiceID)
	if err != nil {
		s.logger.Error("Failed to check existing transaction",
			zap.String("invoice_id", invoiceID),
			zap.Error(err))
		return fmt.Errorf("failed to check existing transaction: %w", err)
	}

	if existingTx != nil {
		s.logger.Info("Credits already allocated for invoice (idempotency check passed)",
			zap.String("invoice_id", invoiceID),
			zap.String("universal_id", universalID.String()),
			zap.String("existing_tx_id", fmt.Sprintf("%d", existingTx.ID)))
		return nil
	} else {
		s.logger.Info("No existing transaction found, proceeding with credit allocation")
	}

	// Allocate credits
	amount := decimal.NewFromInt(int64(creditsPerCycle))
	description := fmt.Sprintf("Credit allocation for %s subscription payment", productName)

	s.logger.Info("CALLING creditRepo.AllocateCredits with metadata",
		zap.String("universal_id", universalID.String()),
		zap.String("amount", amount.String()),
		zap.String("description", description),
		zap.String("reference_id", invoiceID))

	balance, transaction, err := s.creditRepo.AllocateCredits(ctx, universalID, amount, description, invoiceID)
	if err != nil {
		s.logger.Error("CREDIT ALLOCATION WITH METADATA FAILED IN REPOSITORY",
			zap.String("universal_id", universalID.String()),
			zap.String("invoice_id", invoiceID),
			zap.String("amount", amount.String()),
			zap.Error(err))
		return fmt.Errorf("failed to allocate credits: %w", err)
	}

	s.logger.Info("CREDIT ALLOCATION WITH METADATA SUCCESSFUL",
		zap.String("universal_id", universalID.String()),
		zap.String("invoice_id", invoiceID),
		zap.Int("credits", creditsPerCycle),
		zap.String("product", productName),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("transaction_id", fmt.Sprintf("%d", transaction.ID)))

	s.logger.Info("=== AllocateCreditsWithMetadata END ===")
	return nil
}

// GetBalance retrieves the current credit balance for a user
func (s *CreditService) GetBalance(ctx context.Context, universalID uuid.UUID) (*model.UserCreditBalance, error) {
	balance, err := s.creditRepo.GetBalance(ctx, universalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// UseCredits deducts credits for a specific feature
func (s *CreditService) UseCredits(ctx context.Context, universalID uuid.UUID, amount decimal.Decimal, featureName string, description string, usageMetadata []byte, idempotencyKey *uuid.UUID) (*model.CreditTransaction, error) {
	// For now, we'll use the existing UseCredits without idempotency key support
	// TODO: Add idempotency key support to repository layer
	
	// First get the current balance to provide in error if insufficient
	currentBalance, err := s.creditRepo.GetBalance(ctx, universalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	balance, transaction, err := s.creditRepo.UseCredits(ctx, universalID, amount, description, featureName)
	if err != nil {
		// Check if it's an insufficient balance error
		if strings.Contains(err.Error(), "insufficient credit balance") {
			return nil, customErr.NewInsufficientBalanceError(amount, currentBalance.CurrentBalance)
		}
		return nil, fmt.Errorf("failed to use credits: %w", err)
	}

	// Log the successful usage
	s.logger.Info("Credits used successfully",
		zap.String("universal_id", universalID.String()),
		zap.String("amount", amount.String()),
		zap.String("feature", featureName),
		zap.String("balance_after", balance.CurrentBalance.String()),
		zap.Int64("transaction_id", transaction.ID))

	return transaction, nil
}
