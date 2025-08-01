package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/repository"
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
func (s *CreditService) AllocateCreditsForPayment(ctx context.Context, userID uuid.UUID, invoiceID string, subscriptionID string, stripePriceID string) error {
	s.logger.Info("=== AllocateCreditsForPayment START ===",
		zap.String("user_id", userID.String()),
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
			zap.String("user_id", userID.String()),
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
		zap.String("user_id", userID.String()),
		zap.String("amount", amount.String()),
		zap.String("description", description),
		zap.String("reference_id", invoiceID))

	balance, transaction, err := s.creditRepo.AllocateCredits(ctx, userID, amount, description, invoiceID)
	if err != nil {
		s.logger.Error("CREDIT ALLOCATION FAILED IN REPOSITORY",
			zap.String("user_id", userID.String()),
			zap.String("invoice_id", invoiceID),
			zap.String("amount", amount.String()),
			zap.Error(err))
		return fmt.Errorf("failed to allocate credits: %w", err)
	}

	s.logger.Info("CREDIT ALLOCATION SUCCESSFUL",
		zap.String("user_id", userID.String()),
		zap.String("invoice_id", invoiceID),
		zap.String("subscription_id", subscriptionID),
		zap.Int("credits", plan.CreditsPerCycle),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("transaction_id", fmt.Sprintf("%d", transaction.ID)))

	s.logger.Info("=== AllocateCreditsForPayment END ===")
	return nil
}

// AllocateCreditsWithMetadata allocates credits based on product metadata
func (s *CreditService) AllocateCreditsWithMetadata(ctx context.Context, userID uuid.UUID, invoiceID string, creditsPerCycle int, productName string) error {
	s.logger.Info("=== AllocateCreditsWithMetadata START ===",
		zap.String("user_id", userID.String()),
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
			zap.String("user_id", userID.String()),
			zap.String("existing_tx_id", fmt.Sprintf("%d", existingTx.ID)))
		return nil
	} else {
		s.logger.Info("No existing transaction found, proceeding with credit allocation")
	}

	// Allocate credits
	amount := decimal.NewFromInt(int64(creditsPerCycle))
	description := fmt.Sprintf("Credit allocation for %s subscription payment", productName)

	s.logger.Info("CALLING creditRepo.AllocateCredits with metadata",
		zap.String("user_id", userID.String()),
		zap.String("amount", amount.String()),
		zap.String("description", description),
		zap.String("reference_id", invoiceID))

	balance, transaction, err := s.creditRepo.AllocateCredits(ctx, userID, amount, description, invoiceID)
	if err != nil {
		s.logger.Error("CREDIT ALLOCATION WITH METADATA FAILED IN REPOSITORY",
			zap.String("user_id", userID.String()),
			zap.String("invoice_id", invoiceID),
			zap.String("amount", amount.String()),
			zap.Error(err))
		return fmt.Errorf("failed to allocate credits: %w", err)
	}

	s.logger.Info("CREDIT ALLOCATION WITH METADATA SUCCESSFUL",
		zap.String("user_id", userID.String()),
		zap.String("invoice_id", invoiceID),
		zap.Int("credits", creditsPerCycle),
		zap.String("product", productName),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("transaction_id", fmt.Sprintf("%d", transaction.ID)))

	s.logger.Info("=== AllocateCreditsWithMetadata END ===")
	return nil
}

// GetBalance retrieves the current credit balance for a user
func (s *CreditService) GetBalance(ctx context.Context, userID uuid.UUID) (*model.UserCreditBalance, error) {
	balance, err := s.creditRepo.GetBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// UseCredits deducts credits for a specific feature
func (s *CreditService) UseCredits(ctx context.Context, userID uuid.UUID, amount int, featureName string, description string) error {
	decimalAmount := decimal.NewFromInt(int64(amount))

	_, _, err := s.creditRepo.UseCredits(ctx, userID, decimalAmount, description, featureName)
	if err != nil {
		return fmt.Errorf("failed to use credits: %w", err)
	}

	return nil
}
