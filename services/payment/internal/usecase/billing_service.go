package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/crypto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/provider/toss"
	"go.uber.org/zap"
)

type BillingService struct {
	billingKeyRepo repository.BillingKeyRepository
	paymentRepo    repository.PaymentRepository
	tossProvider   *toss.TossProvider
	encryptService crypto.EncryptionService
	creditService  *CreditService
	logger         *zap.Logger
}

func NewBillingService(
	billingKeyRepo repository.BillingKeyRepository,
	paymentRepo repository.PaymentRepository,
	tossProvider *toss.TossProvider,
	encryptService crypto.EncryptionService,
	creditService *CreditService,
	logger *zap.Logger,
) *BillingService {
	return &BillingService{
		billingKeyRepo: billingKeyRepo,
		paymentRepo:    paymentRepo,
		tossProvider:   tossProvider,
		encryptService: encryptService,
		creditService:  creditService,
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

type ChargeBillingKeyResult struct {
	OrderID          string
	PaymentKey       string
	TransactionKey   string
	Status           string
	Amount           int64
	ApprovedAt       *time.Time
	CreditsAllocated int
}

func (s *BillingService) ChargeBillingKey(
	ctx context.Context,
	universalID uuid.UUID,
	billingKeyID int64,
	amount int64,
	orderName string,
	planID string,
	serviceProvider string,
	ipAddress string,
	userAgent string,
) (*ChargeBillingKeyResult, error) {
	billingKey, err := s.billingKeyRepo.GetByID(ctx, billingKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing key: %w", err)
	}
	if billingKey == nil {
		return nil, fmt.Errorf("billing key not found")
	}
	if billingKey.UniversalID != universalID {
		return nil, fmt.Errorf("unauthorized: billing key does not belong to user")
	}
	if !billingKey.IsActive {
		return nil, fmt.Errorf("billing key is not active")
	}

	decryptedBillingKey, err := s.encryptService.Decrypt(billingKey.EncryptedBillingKey, billingKey.EncryptionIV)
	if err != nil {
		s.logger.Error("failed to decrypt billing key",
			zap.Int64("billing_key_id", billingKeyID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to decrypt billing key: %w", err)
	}

	orderID := s.generateOrderID()

	payment := &entity.Payment{
		UniversalID:   universalID.String(),
		TransactionID: orderID,
		Amount:        float64(amount),
		Currency:      "KRW",
		Status:        entity.PaymentStatusPending,
		Method:        entity.PaymentMethodCard,
		Description:   orderName,
		Metadata: map[string]interface{}{
			"plan_id":          planID,
			"service_provider": serviceProvider,
			"customer_key":     billingKey.CustomerKey,
			"billing_key_id":   billingKeyID,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.paymentRepo.CreateOneTimePayment(ctx, payment); err != nil {
		s.logger.Error("failed to create payment record",
			zap.String("order_id", orderID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	chargeResp, err := s.tossProvider.ChargeBillingKey(ctx, &provider.ChargeBillingKeyRequest{
		BillingKey:  decryptedBillingKey,
		CustomerKey: billingKey.CustomerKey,
		Amount:      amount,
		OrderID:     orderID,
		OrderName:   orderName,
	})
	if err != nil {
		s.paymentRepo.UpdatePaymentAfterConfirm(ctx, orderID, map[string]interface{}{
			"status": string(entity.PaymentStatusFailed),
		})
		s.logger.Error("failed to charge billing key",
			zap.Int64("billing_key_id", billingKeyID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to charge billing key: %w", err)
	}

	updates := map[string]interface{}{
		"status":                     string(entity.PaymentStatusCompleted),
		"provider_payment_intent_id": chargeResp.PaymentKey,
		"provider_charge_id":         chargeResp.TransactionKey,
	}
	if chargeResp.ApprovedAt != nil {
		updates["paid_at"] = chargeResp.ApprovedAt
	}

	if err := s.paymentRepo.UpdatePaymentAfterConfirm(ctx, orderID, updates); err != nil {
		s.logger.Error("failed to update payment after charge",
			zap.String("order_id", orderID),
			zap.Error(err))
	}

	s.billingKeyRepo.CreateAccessLog(ctx, &model.BillingKeyAccessLog{
		BillingKeyID: billingKeyID,
		AccessType:   "charge",
		AccessorID:   universalID.String(),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Purpose:      "billing_key_charge",
	})

	s.logger.Info("billing key charged successfully",
		zap.Int64("billing_key_id", billingKeyID),
		zap.String("order_id", orderID),
		zap.String("payment_key", chargeResp.PaymentKey))

	// Allocate credits synchronously (billing key payments don't trigger webhooks)
	var creditsAllocated int
	if chargeResp.Status == "DONE" && s.creditService != nil {
		allocated, err := s.creditService.AllocateCreditsForPayment(
			ctx,
			universalID,
			orderID,
			"",
			planID,
			serviceProvider,
		)
		if err != nil {
			s.logger.Error("failed to allocate credits after billing charge",
				zap.String("order_id", orderID),
				zap.String("plan_id", planID),
				zap.Error(err))
		} else {
			creditsAllocated = allocated
			s.logger.Info("credits allocated for billing charge",
				zap.String("order_id", orderID),
				zap.Int("credits", creditsAllocated))
		}
	}

	return &ChargeBillingKeyResult{
		OrderID:          orderID,
		PaymentKey:       chargeResp.PaymentKey,
		TransactionKey:   chargeResp.TransactionKey,
		Status:           chargeResp.Status,
		Amount:           chargeResp.Amount,
		ApprovedAt:       chargeResp.ApprovedAt,
		CreditsAllocated: creditsAllocated,
	}, nil
}

func (s *BillingService) generateOrderID() string {
	return fmt.Sprintf("ORDER_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
}
