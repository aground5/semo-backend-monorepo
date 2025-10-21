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
	serviceProvider  string
}

// NewCreditService creates a new credit service instance
func NewCreditService(
	creditRepo domainRepo.CreditRepository,
	subscriptionRepo domainRepo.SubscriptionRepository,
	planRepo repository.PlanRepository,
	logger *zap.Logger,
	serviceProvider string,
) *CreditService {
	if serviceProvider == "" {
		logger.Error("CreditService initialized without service provider")
	}
	return &CreditService{
		creditRepo:       creditRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
		serviceProvider:  serviceProvider,
	}
}

// AllocateCreditsForPayment allocates credits based on a successful payment
// Returns the number of credits newly allocated. When 0 is returned without error,
// the allocation was already processed earlier (idempotency).
func (s *CreditService) AllocateCreditsForPayment(ctx context.Context, universalID uuid.UUID, invoiceID string, subscriptionID string, stripePriceID string, serviceProviderOverride string) (int, error) {
	s.logger.Info("=== AllocateCreditsForPayment START ===",
		zap.String("universal_id", universalID.String()),
		zap.String("invoice_id", invoiceID),
		zap.String("subscription_id", subscriptionID),
		zap.String("price_id", stripePriceID),
		zap.String("service_provider", serviceProviderOverride))

	// First check if credits were already allocated for this invoice (idempotency)
	s.logger.Info("Checking for existing transaction", zap.String("invoice_id", invoiceID))
	existingTx, err := s.creditRepo.GetTransactionByReference(ctx, invoiceID)
	if err != nil {
		s.logger.Error("Failed to check existing transaction",
			zap.String("invoice_id", invoiceID),
			zap.Error(err))
		return 0, fmt.Errorf("failed to check existing transaction: %w", err)
	}

	if existingTx != nil {
		s.logger.Info("Credits already allocated for invoice (idempotency check passed)",
			zap.String("invoice_id", invoiceID),
			zap.String("universal_id", universalID.String()),
			zap.String("existing_tx_id", fmt.Sprintf("%d", existingTx.ID)))
		return 0, nil
	} else {
		s.logger.Info("No existing transaction found, proceeding with credit allocation")
	}

	// Get the payment plan to determine credits
	s.logger.Info("Looking up payment plan by price ID", zap.String("price_id", stripePriceID))
	plan, err := s.planRepo.GetByPriceID(ctx, stripePriceID)
	if err != nil {
		s.logger.Error("Failed to get payment plan by price ID",
			zap.String("price_id", stripePriceID),
			zap.Error(err))
		return 0, fmt.Errorf("failed to get payment plan: %w", err)
	}

	if plan == nil {
		s.logger.Warn("No payment plan found by price ID, attempting product ID lookup",
			zap.String("identifier", stripePriceID))

		plansByProduct, err := s.planRepo.GetByProductID(ctx, stripePriceID)
		if err != nil {
			s.logger.Error("Failed to get payment plan by product ID",
				zap.String("product_id", stripePriceID),
				zap.Error(err))
			return 0, fmt.Errorf("failed to get payment plan by product ID: %w", err)
		}
		if len(plansByProduct) == 0 {
			s.logger.Error("No payment plan found by price or product ID",
				zap.String("identifier", stripePriceID))
			return 0, fmt.Errorf("payment plan not found for identifier: %s", stripePriceID)
		}

		plan = plansByProduct[0]
		s.logger.Info("Found payment plan via product ID lookup",
			zap.String("plan_name", plan.DisplayName),
			zap.Int("credits_per_cycle", plan.CreditsPerCycle),
			zap.String("provider_product_id", plan.ProviderProductID))
	} else {
		s.logger.Info("Found payment plan",
			zap.String("plan_name", plan.DisplayName),
			zap.Int("credits_per_cycle", plan.CreditsPerCycle),
			zap.String("provider_price_id", plan.ProviderPriceID))
	}

	// Allocate credits
	amount := decimal.NewFromInt(int64(plan.CreditsPerCycle))
	description := fmt.Sprintf("Credit allocation for %s subscription", plan.DisplayName)

	s.logger.Info("CALLING creditRepo.AllocateCredits",
		zap.String("universal_id", universalID.String()),
		zap.String("amount", amount.String()),
		zap.String("description", description),
		zap.String("reference_id", invoiceID))

	serviceProvider := s.serviceProvider
	if serviceProviderOverride != "" {
		serviceProvider = serviceProviderOverride
	}

	s.logger.Info("Resolved service provider for credit allocation",
		zap.String("universal_id", universalID.String()),
		zap.String("service_provider", serviceProvider))

	balance, transaction, err := s.creditRepo.AllocateCredits(ctx, universalID, serviceProvider, amount, description, invoiceID)
	if err != nil {
		s.logger.Error("CREDIT ALLOCATION FAILED IN REPOSITORY",
			zap.String("universal_id", universalID.String()),
			zap.String("invoice_id", invoiceID),
			zap.String("amount", amount.String()),
			zap.Error(err))
		return 0, fmt.Errorf("failed to allocate credits: %w", err)
	}

	s.logger.Info("CREDIT ALLOCATION SUCCESSFUL",
		zap.String("universal_id", universalID.String()),
		zap.String("invoice_id", invoiceID),
		zap.String("subscription_id", subscriptionID),
		zap.Int("credits", plan.CreditsPerCycle),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("transaction_id", fmt.Sprintf("%d", transaction.ID)))

	s.logger.Info("=== AllocateCreditsForPayment END ===")
	return plan.CreditsPerCycle, nil
}

// AllocateCreditsWithMetadata allocates credits based on product metadata
// Returns the number of credits newly allocated. When 0 is returned without error,
// the allocation was already processed earlier (idempotency).
func (s *CreditService) AllocateCreditsWithMetadata(ctx context.Context, universalID uuid.UUID, invoiceID string, creditsPerCycle int, productName string) (int, error) {
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
		return 0, fmt.Errorf("failed to check existing transaction: %w", err)
	}

	if existingTx != nil {
		s.logger.Info("Credits already allocated for invoice (idempotency check passed)",
			zap.String("invoice_id", invoiceID),
			zap.String("universal_id", universalID.String()),
			zap.String("existing_tx_id", fmt.Sprintf("%d", existingTx.ID)))
		return 0, nil
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

	balance, transaction, err := s.creditRepo.AllocateCredits(ctx, universalID, s.serviceProvider, amount, description, invoiceID)
	if err != nil {
		s.logger.Error("CREDIT ALLOCATION WITH METADATA FAILED IN REPOSITORY",
			zap.String("universal_id", universalID.String()),
			zap.String("invoice_id", invoiceID),
			zap.String("amount", amount.String()),
			zap.Error(err))
		return 0, fmt.Errorf("failed to allocate credits: %w", err)
	}

	s.logger.Info("CREDIT ALLOCATION WITH METADATA SUCCESSFUL",
		zap.String("universal_id", universalID.String()),
		zap.String("invoice_id", invoiceID),
		zap.Int("credits", creditsPerCycle),
		zap.String("product", productName),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("transaction_id", fmt.Sprintf("%d", transaction.ID)))

	s.logger.Info("=== AllocateCreditsWithMetadata END ===")
	return creditsPerCycle, nil
}

// AllocateCreditsManual allocates a fixed number of credits using the provided details.
// This supports system-driven adjustments like Supabase onboarding credits.
func (s *CreditService) AllocateCreditsManual(ctx context.Context, universalID uuid.UUID, serviceProvider string, credits int, description string, referenceID string) (*model.UserCreditBalance, *model.CreditTransaction, error) {
	if credits <= 0 {
		return nil, nil, fmt.Errorf("credits must be positive for manual allocation")
	}

	resolvedProvider := strings.TrimSpace(serviceProvider)
	if resolvedProvider == "" {
		resolvedProvider = s.serviceProvider
	}

	amount := decimal.NewFromInt(int64(credits))

	s.logger.Info("Allocating manual credits",
		zap.String("universal_id", universalID.String()),
		zap.String("service_provider", resolvedProvider),
		zap.Int("credits", credits),
		zap.String("reference_id", referenceID),
		zap.String("description", description))

	balance, transaction, err := s.creditRepo.AllocateCredits(ctx, universalID, resolvedProvider, amount, description, referenceID)
	if err != nil {
		s.logger.Error("Manual credit allocation failed",
			zap.String("universal_id", universalID.String()),
			zap.String("service_provider", resolvedProvider),
			zap.Int("credits", credits),
			zap.String("reference_id", referenceID),
			zap.Error(err))
		return nil, nil, fmt.Errorf("failed to allocate manual credits: %w", err)
	}

	s.logger.Info("Manual credit allocation successful",
		zap.String("universal_id", universalID.String()),
		zap.String("service_provider", resolvedProvider),
		zap.String("new_balance", balance.CurrentBalance.String()),
		zap.String("reference_id", referenceID),
		zap.Int64("transaction_id", transaction.ID))

	return balance, transaction, nil
}

// GetBalance retrieves the current credit balance for a user
func (s *CreditService) GetBalance(ctx context.Context, universalID uuid.UUID) (*model.UserCreditBalance, error) {
	return s.GetBalanceForProvider(ctx, universalID, "")
}

// GetBalanceForProvider retrieves the current credit balance for a user and provider.
func (s *CreditService) GetBalanceForProvider(ctx context.Context, universalID uuid.UUID, providerOverride string) (*model.UserCreditBalance, error) {
	provider := strings.TrimSpace(providerOverride)
	if provider == "" {
		provider = s.serviceProvider
	}

	balance, err := s.creditRepo.GetBalance(ctx, universalID, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// UseCredits deducts credits for a specific feature
func (s *CreditService) UseCredits(ctx context.Context, universalID uuid.UUID, serviceProvider string, amount decimal.Decimal, featureName string, description string, usageMetadata []byte, idempotencyKey *uuid.UUID) (*model.CreditTransaction, error) {
	// For now, we'll use the existing UseCredits without idempotency key support
	// TODO: Add idempotency key support to repository layer

	provider := serviceProvider
	if provider == "" {
		provider = s.serviceProvider
		s.logger.Warn("UseCredits called without service provider; falling back to default",
			zap.String("universal_id", universalID.String()))
	}

	// First get the current balance to provide in error if insufficient
	currentBalance, err := s.creditRepo.GetBalance(ctx, universalID, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	balance, transaction, err := s.creditRepo.UseCredits(ctx, universalID, provider, amount, description, featureName)
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
		zap.String("service_provider", provider),
		zap.String("amount", amount.String()),
		zap.String("feature", featureName),
		zap.String("balance_after", balance.CurrentBalance.String()),
		zap.Int64("transaction_id", transaction.ID))

	return transaction, nil
}
