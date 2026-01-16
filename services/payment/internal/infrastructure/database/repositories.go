package database

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	Payment               domainRepo.PaymentRepository
	Subscription          domainRepo.SubscriptionRepository
	CustomerMapping       domainRepo.CustomerMappingRepository
	Credit                domainRepo.CreditRepository
	CreditTransaction     domainRepo.CreditTransactionRepository
	Webhook               repository.WebhookRepository
	Plan                  repository.PlanRepository
	WorkspaceVerification domainRepo.WorkspaceVerificationRepository
	BillingKey            domainRepo.BillingKeyRepository
}

// NewRepositories creates new repository instances with database connection
func NewRepositories(db *gorm.DB, supabaseConfig *config.SupabaseConfig, logger *zap.Logger) *Repositories {
	// Create customer mapping repository first as it's a dependency
	customerMappingRepo := repository.NewCustomerMappingRepository(db)

	// Create credit repository as it's a dependency for subscription repository
	creditRepo := repository.NewCreditRepository(db, logger)

	// Create workspace verification repository
	workspaceVerificationRepo := repository.NewSupabaseWorkspaceVerificationRepository(
		supabaseConfig.ProjectURL,
		supabaseConfig.APIKey,
		supabaseConfig.JWTSecret,
		logger,
	)

	return &Repositories{
		Payment:               repository.NewPaymentRepository(db, logger),
		Subscription:          repository.NewSubscriptionRepository(db, logger, customerMappingRepo, creditRepo),
		CustomerMapping:       customerMappingRepo,
		Credit:                creditRepo,
		CreditTransaction:     repository.NewCreditTransactionRepository(db, logger),
		Webhook:               repository.NewWebhookRepository(db, logger),
		Plan:                  repository.NewPlanRepository(db, logger),
		WorkspaceVerification: workspaceVerificationRepo,
		BillingKey:            repository.NewBillingKeyRepository(db, logger),
	}
}
