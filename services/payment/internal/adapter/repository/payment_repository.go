package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type paymentRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPaymentRepository(db *gorm.DB, logger *zap.Logger) repository.PaymentRepository {
	return &paymentRepository{
		db:     db,
		logger: logger,
	}
}

func (r *paymentRepository) Create(ctx context.Context, payment *entity.Payment) error {
	// Convert entity to model
	userID, err := uuid.Parse(payment.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	paymentModel := &model.Payment{
		UserID:                userID,
		StripePaymentIntentID: &payment.TransactionID,
		AmountCents:           int(payment.Amount * 100), // Convert to cents
		Currency:              payment.Currency,
		Status:                string(payment.Status),
		PaymentMethodType:     (*string)(&payment.Method),
	}

	err = r.db.WithContext(ctx).Create(paymentModel).Error
	if err != nil {
		r.logger.Error("Failed to create payment",
			zap.String("user_id", payment.UserID),
			zap.Error(err))
		return fmt.Errorf("failed to create payment: %w", err)
	}

	// Set the ID back to the entity
	payment.ID = fmt.Sprintf("%d", paymentModel.ID)
	return nil
}

func (r *paymentRepository) GetByID(ctx context.Context, id string) (*entity.Payment, error) {
	var payment model.Payment

	err := r.db.WithContext(ctx).First(&payment, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get payment by ID",
			zap.String("id", id),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return r.modelToEntity(&payment), nil
}

func (r *paymentRepository) GetByUserID(ctx context.Context, userID string) ([]*entity.Payment, error) {
	var payments []model.Payment

	uuid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	err = r.db.WithContext(ctx).
		Where("user_id = ?", uuid).
		Order("created_at DESC").
		Find(&payments).Error

	if err != nil {
		r.logger.Error("Failed to get payments by user ID",
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get payments: %w", err)
	}

	entities := make([]*entity.Payment, len(payments))
	for i, p := range payments {
		entities[i] = r.modelToEntity(&p)
	}

	return entities, nil
}

func (r *paymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*entity.Payment, error) {
	var payment model.Payment

	err := r.db.WithContext(ctx).
		Where("stripe_payment_intent_id = ?", transactionID).
		First(&payment).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get payment by transaction ID",
			zap.String("transaction_id", transactionID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return r.modelToEntity(&payment), nil
}

func (r *paymentRepository) Update(ctx context.Context, payment *entity.Payment) error {
	// Parse payment ID
	var paymentID int64
	_, err := fmt.Sscanf(payment.ID, "%d", &paymentID)
	if err != nil {
		return fmt.Errorf("invalid payment ID: %w", err)
	}

	// Update fields
	updates := map[string]interface{}{
		"status":     string(payment.Status),
		"updated_at": gorm.Expr("NOW()"),
	}

	if payment.Description != "" {
		updates["failure_message"] = payment.Description
	}

	err = r.db.WithContext(ctx).
		Model(&model.Payment{}).
		Where("id = ?", paymentID).
		Updates(updates).Error

	if err != nil {
		r.logger.Error("Failed to update payment",
			zap.String("id", payment.ID),
			zap.Error(err))
		return fmt.Errorf("failed to update payment: %w", err)
	}

	return nil
}

func (r *paymentRepository) Delete(ctx context.Context, id string) error {
	// Parse payment ID
	var paymentID int64
	_, err := fmt.Sscanf(id, "%d", &paymentID)
	if err != nil {
		return fmt.Errorf("invalid payment ID: %w", err)
	}

	// Soft delete
	err = r.db.WithContext(ctx).
		Delete(&model.Payment{}, paymentID).Error

	if err != nil {
		r.logger.Error("Failed to delete payment",
			zap.String("id", id),
			zap.Error(err))
		return fmt.Errorf("failed to delete payment: %w", err)
	}

	return nil
}

func (r *paymentRepository) List(ctx context.Context, limit, offset int) ([]*entity.Payment, error) {
	var payments []model.Payment

	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&payments).Error
	if err != nil {
		r.logger.Error("Failed to list payments",
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	entities := make([]*entity.Payment, len(payments))
	for i, p := range payments {
		entities[i] = r.modelToEntity(&p)
	}

	return entities, nil
}

// modelToEntity converts database model to domain entity
func (r *paymentRepository) modelToEntity(m *model.Payment) *entity.Payment {
	if m == nil {
		return nil
	}

	e := &entity.Payment{
		ID:        fmt.Sprintf("%d", m.ID),
		UserID:    m.UserID.String(),
		Amount:    float64(m.AmountCents) / 100, // Convert from cents
		Currency:  m.Currency,
		Status:    entity.PaymentStatus(m.Status),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.StripePaymentIntentID != nil {
		e.TransactionID = *m.StripePaymentIntentID
	}

	if m.PaymentMethodType != nil {
		e.Method = entity.PaymentMethod(*m.PaymentMethodType)
	}

	// Convert metadata if needed
	e.Metadata = make(map[string]interface{})
	if m.StripePaymentData != nil {
		for k, v := range m.StripePaymentData {
			e.Metadata[k] = v
		}
	}

	return e
}
