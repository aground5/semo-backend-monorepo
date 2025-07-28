package database

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/repository"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	Payment         domainRepo.PaymentRepository
	Subscription    domainRepo.SubscriptionRepository
	CustomerMapping domainRepo.CustomerMappingRepository
	Webhook         repository.WebhookRepository
	Plan            repository.PlanRepository
}

// NewRepositories creates new repository instances with database connection
func NewRepositories(db *gorm.DB, logger *zap.Logger) *Repositories {
	return &Repositories{
		Payment:         repository.NewPaymentRepository(db, logger),
		Subscription:    repository.NewSubscriptionRepository(db, logger),
		CustomerMapping: repository.NewCustomerMappingRepository(db),
		Webhook:         repository.NewWebhookRepository(db, logger),
		Plan:            repository.NewPlanRepository(db, logger),
	}
}
