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
	universalID, err := uuid.Parse(payment.UniversalID)
	if err != nil {
		return fmt.Errorf("invalid universal ID: %w", err)
	}

	paymentModel := &model.Payment{
		UniversalID:           universalID,
		ProviderPaymentIntentID: &payment.TransactionID,
		AmountCents:           int(payment.Amount * 100), // Convert to cents
		Currency:              payment.Currency,
		Status:                string(payment.Status),
		PaymentMethodType:     (*string)(&payment.Method),
	}

	err = r.db.WithContext(ctx).Create(paymentModel).Error
	if err != nil {
		r.logger.Error("Failed to create payment",
			zap.String("universal_id", payment.UniversalID),
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

func (r *paymentRepository) GetByUniversalID(ctx context.Context, universalID string, page, limit int) ([]*entity.Payment, int64, error) {
	var payments []model.Payment
	var total int64

	uuid, err := uuid.Parse(universalID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid universal ID: %w", err)
	}

	// Get total count
	err = r.db.WithContext(ctx).
		Model(&model.Payment{}).
		Where("universal_id = ?", uuid).
		Count(&total).Error
	
	if err != nil {
		r.logger.Error("Failed to count payments by universal ID",
			zap.String("universal_id", universalID),
			zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	// If no records, return early
	if total == 0 {
		return []*entity.Payment{}, 0, nil
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get paginated records
	err = r.db.WithContext(ctx).
		Where("universal_id = ?", uuid).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&payments).Error

	if err != nil {
		r.logger.Error("Failed to get payments by universal ID",
			zap.String("universal_id", universalID),
			zap.Int("page", page),
			zap.Int("limit", limit),
			zap.Error(err))
		return nil, 0, fmt.Errorf("failed to get payments: %w", err)
	}

	entities := make([]*entity.Payment, len(payments))
	for i, p := range payments {
		entities[i] = r.modelToEntity(&p)
	}

	return entities, total, nil
}

func (r *paymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*entity.Payment, error) {
	var payment model.Payment

	err := r.db.WithContext(ctx).
		Where("provider_payment_intent_id = ?", transactionID).
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

// CreateOneTimePayment creates a new one-time payment record
func (r *paymentRepository) CreateOneTimePayment(ctx context.Context, payment *entity.Payment) error {
	universalID, err := uuid.Parse(payment.UniversalID)
	if err != nil {
		return fmt.Errorf("invalid universal ID: %w", err)
	}

	paymentModel := &model.Payment{
		UniversalID:            universalID,
		ProviderInvoiceID:      &payment.TransactionID, // Use order ID
		AmountCents:            int(payment.Amount),    // Already in smallest unit
		Currency:               payment.Currency,
		Status:                 string(payment.Status),
		ProviderPaymentData:    payment.Metadata,
	}

	err = r.db.WithContext(ctx).Create(paymentModel).Error
	if err != nil {
		r.logger.Error("Failed to create one-time payment",
			zap.String("universal_id", payment.UniversalID),
			zap.String("order_id", payment.TransactionID),
			zap.Error(err))
		return fmt.Errorf("failed to create one-time payment: %w", err)
	}

	payment.ID = fmt.Sprintf("%d", paymentModel.ID)
	return nil
}

// GetByOrderID retrieves a payment by order ID
func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error) {
	var payment model.Payment

	err := r.db.WithContext(ctx).
		Where("provider_invoice_id = ?", orderID).
		First(&payment).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get payment by order ID",
			zap.String("order_id", orderID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return r.modelToEntity(&payment), nil
}

// UpdatePaymentAfterConfirm updates payment after provider confirmation
func (r *paymentRepository) UpdatePaymentAfterConfirm(ctx context.Context, orderID string, updates map[string]interface{}) error {
	updates["updated_at"] = gorm.Expr("NOW()")

	err := r.db.WithContext(ctx).
		Model(&model.Payment{}).
		Where("provider_invoice_id = ?", orderID).
		Updates(updates).Error

	if err != nil {
		r.logger.Error("Failed to update payment after confirmation",
			zap.String("order_id", orderID),
			zap.Error(err))
		return fmt.Errorf("failed to update payment: %w", err)
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
		ID:          fmt.Sprintf("%d", m.ID),
		UniversalID: m.UniversalID.String(),
		Amount:      float64(m.AmountCents), // Convert from cents
		Currency:  m.Currency,
		Status:    entity.PaymentStatus(m.Status),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.ProviderPaymentIntentID != nil {
		e.TransactionID = *m.ProviderPaymentIntentID
	}

	if m.PaymentMethodType != nil {
		e.Method = entity.PaymentMethod(*m.PaymentMethodType)
	}

	// Convert metadata if needed
	e.Metadata = make(map[string]interface{})
	if m.ProviderPaymentData != nil {
		for k, v := range m.ProviderPaymentData {
			e.Metadata[k] = v
		}
	}

	return e
}
