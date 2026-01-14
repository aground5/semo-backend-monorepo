package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type billingKeyRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBillingKeyRepository(db *gorm.DB, logger *zap.Logger) domainRepo.BillingKeyRepository {
	return &billingKeyRepository{db: db, logger: logger}
}

func (r *billingKeyRepository) Create(ctx context.Context, billingKey *model.BillingKey) error {
	if err := r.db.WithContext(ctx).Create(billingKey).Error; err != nil {
		r.logger.Error("failed to create billing key",
			zap.String("universal_id", billingKey.UniversalID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to create billing key: %w", err)
	}
	return nil
}

func (r *billingKeyRepository) GetByID(ctx context.Context, id int64) (*model.BillingKey, error) {
	var billingKey model.BillingKey
	err := r.db.WithContext(ctx).First(&billingKey, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("failed to get billing key by id",
			zap.Int64("id", id),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get billing key: %w", err)
	}
	return &billingKey, nil
}

func (r *billingKeyRepository) GetByCustomerKey(ctx context.Context, customerKey string) (*model.BillingKey, error) {
	var billingKey model.BillingKey
	err := r.db.WithContext(ctx).Where("customer_key = ?", customerKey).First(&billingKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("failed to get billing key by customer key",
			zap.String("customer_key", customerKey),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get billing key: %w", err)
	}
	return &billingKey, nil
}

func (r *billingKeyRepository) GetActiveByUniversalID(ctx context.Context, universalID uuid.UUID) ([]*model.BillingKey, error) {
	var billingKeys []*model.BillingKey
	err := r.db.WithContext(ctx).
		Where("universal_id = ? AND is_active = ?", universalID, true).
		Order("created_at DESC").
		Find(&billingKeys).Error
	if err != nil {
		r.logger.Error("failed to get active billing keys",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get billing keys: %w", err)
	}
	return billingKeys, nil
}

func (r *billingKeyRepository) Deactivate(ctx context.Context, id int64) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.BillingKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":      false,
			"deactivated_at": now,
			"updated_at":     now,
		})

	if result.Error != nil {
		r.logger.Error("failed to deactivate billing key",
			zap.Int64("id", id),
			zap.Error(result.Error))
		return fmt.Errorf("failed to deactivate billing key: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("billing key not found: %d", id)
	}

	return nil
}

func (r *billingKeyRepository) CreateAccessLog(ctx context.Context, log *model.BillingKeyAccessLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		r.logger.Error("failed to create access log",
			zap.Int64("billing_key_id", log.BillingKeyID),
			zap.String("access_type", log.AccessType),
			zap.Error(err))
		return fmt.Errorf("failed to create access log: %w", err)
	}
	return nil
}
