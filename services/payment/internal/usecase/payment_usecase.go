package usecase

import (
	"context"
	"errors"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

type PaymentUsecase struct {
	paymentRepo repository.PaymentRepository
	cacheRepo   repository.CacheRepository
	logger      *zap.Logger
}

func NewPaymentUsecase(
	paymentRepo repository.PaymentRepository,
	cacheRepo repository.CacheRepository,
	logger *zap.Logger,
) *PaymentUsecase {
	return &PaymentUsecase{
		paymentRepo: paymentRepo,
		cacheRepo:   cacheRepo,
		logger:      logger,
	}
}

func (u *PaymentUsecase) CreatePayment(ctx context.Context, payment *entity.Payment) error {
	if payment == nil {
		return errors.New("payment is required")
	}

	if payment.Amount <= 0 {
		return errors.New("invalid payment amount")
	}

	if payment.Currency == "" {
		return errors.New("currency is required")
	}

	return u.paymentRepo.Create(ctx, payment)
}

func (u *PaymentUsecase) GetPayment(ctx context.Context, id string) (*entity.Payment, error) {
	if id == "" {
		return nil, errors.New("payment ID is required")
	}

	return u.paymentRepo.GetByID(ctx, id)
}

func (u *PaymentUsecase) GetPayments(ctx context.Context, universalID string, page, limit int) (*entity.PaginatedPaymentsResponse, error) {
	if universalID == "" {
		return nil, errors.New("user ID is required")
	}

	// Create and validate pagination params
	params := &entity.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	// Get paginated payments from repository
	payments, total, err := u.paymentRepo.GetByUniversalID(ctx, universalID, params.Page, params.Limit)
	if err != nil {
		return nil, err
	}

	// Create pagination metadata
	meta := entity.NewPaginationMeta(params.Page, params.Limit, total)

	// Return paginated response
	return &entity.PaginatedPaymentsResponse{
		Data:       payments,
		Pagination: meta,
	}, nil
}

func (u *PaymentUsecase) UpdatePaymentStatus(ctx context.Context, id string, status entity.PaymentStatus) error {
	payment, err := u.paymentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	payment.Status = status
	return u.paymentRepo.Update(ctx, payment)
}
