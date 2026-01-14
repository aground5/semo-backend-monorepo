package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/crypto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider/toss"
	"go.uber.org/zap"
)

type BillingService struct {
	billingKeyRepo repository.BillingKeyRepository
	tossProvider   *toss.TossProvider
	encryptService crypto.EncryptionService
	logger         *zap.Logger
}

func NewBillingService(
	billingKeyRepo repository.BillingKeyRepository,
	tossProvider *toss.TossProvider,
	encryptService crypto.EncryptionService,
	logger *zap.Logger,
) *BillingService {
	return &BillingService{
		billingKeyRepo: billingKeyRepo,
		tossProvider:   tossProvider,
		encryptService: encryptService,
		logger:         logger,
	}
}

func (s *BillingService) IssueBillingKey(
	ctx context.Context,
	universalID uuid.UUID,
	authKey string,
	customerKey string,
	ipAddress string,
	userAgent string,
) (*model.BillingKey, error) {
	s.logger.Info("IssueBillingKey called",
		zap.String("universal_id", universalID.String()),
		zap.String("customer_key", customerKey),
		zap.String("auth_key_prefix", authKey[:min(len(authKey), 20)]+"..."))

	resp, err := s.tossProvider.IssueBillingKey(ctx, &provider.IssueBillingKeyRequest{
		AuthKey:     authKey,
		CustomerKey: customerKey,
	})
	if err != nil {
		s.logger.Error("failed to issue billing key from toss",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to issue billing key: %w", err)
	}

	encryptedKey, iv, err := s.encryptService.Encrypt(resp.BillingKey)
	if err != nil {
		s.logger.Error("failed to encrypt billing key",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to encrypt billing key: %w", err)
	}

	cardLastFour := ""
	if len(resp.CardNumber) >= 4 {
		cardLastFour = resp.CardNumber[len(resp.CardNumber)-4:]
	}

	billingKey := &model.BillingKey{
		UniversalID:         universalID,
		CustomerKey:         customerKey,
		EncryptedBillingKey: encryptedKey,
		EncryptionIV:        iv,
		CardLastFour:        cardLastFour,
		CardCompany:         resp.CardCompany,
		CardType:            resp.CardType,
		IsActive:            true,
	}

	if err := s.billingKeyRepo.Create(ctx, billingKey); err != nil {
		s.logger.Error("failed to save billing key",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to save billing key: %w", err)
	}

	s.billingKeyRepo.CreateAccessLog(ctx, &model.BillingKeyAccessLog{
		BillingKeyID: billingKey.ID,
		AccessType:   "encrypt",
		AccessorID:   "system",
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Purpose:      "billing_key_issue",
	})

	s.logger.Info("billing key issued successfully",
		zap.String("universal_id", universalID.String()),
		zap.Int64("billing_key_id", billingKey.ID))

	return billingKey, nil
}

func (s *BillingService) GetCards(ctx context.Context, universalID uuid.UUID) ([]*model.BillingKey, error) {
	cards, err := s.billingKeyRepo.GetActiveByUniversalID(ctx, universalID)
	if err != nil {
		s.logger.Error("failed to get cards",
			zap.String("universal_id", universalID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get cards: %w", err)
	}
	return cards, nil
}

func (s *BillingService) DeactivateCard(
	ctx context.Context,
	id int64,
	universalID uuid.UUID,
	ipAddress string,
	userAgent string,
) error {
	billingKey, err := s.billingKeyRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get billing key: %w", err)
	}
	if billingKey == nil {
		return fmt.Errorf("billing key not found")
	}
	if billingKey.UniversalID != universalID {
		return fmt.Errorf("unauthorized: billing key does not belong to user")
	}

	s.billingKeyRepo.CreateAccessLog(ctx, &model.BillingKeyAccessLog{
		BillingKeyID: id,
		AccessType:   "deactivate",
		AccessorID:   universalID.String(),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Purpose:      "user_request",
	})

	if err := s.billingKeyRepo.Deactivate(ctx, id); err != nil {
		s.logger.Error("failed to deactivate billing key",
			zap.Int64("billing_key_id", id),
			zap.Error(err))
		return fmt.Errorf("failed to deactivate card: %w", err)
	}

	s.logger.Info("billing key deactivated",
		zap.Int64("billing_key_id", id),
		zap.String("universal_id", universalID.String()))

	return nil
}
