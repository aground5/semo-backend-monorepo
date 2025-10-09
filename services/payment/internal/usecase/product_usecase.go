package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

// ProductUseCase handles one-time payment operations
type ProductUseCase struct {
	paymentRepo repository.PaymentRepository
	logger      *zap.Logger
}

// NewProductUseCase creates a new ProductUseCase instance
func NewProductUseCase(
	paymentRepo repository.PaymentRepository,
	logger *zap.Logger,
) *ProductUseCase {
	return &ProductUseCase{
		paymentRepo: paymentRepo,
		logger:      logger,
	}
}

// CreateProductRequest represents a request to create a payment
type CreateProductRequest struct {
	UniversalID string                 `json:"universal_id"`
	Amount      int64                  `json:"amount"`
	Currency    string                 `json:"currency"`
	OrderName   string                 `json:"order_name"`
	CustomerKey string                 `json:"customer_key"`
	PlanID      string                 `json:"plan_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateProductResponse represents the response from payment creation
type CreateProductResponse struct {
	OrderID      string                 `json:"order_id"`
	PaymentKey   string                 `json:"payment_key,omitempty"`
	ClientSecret string                 `json:"client_secret,omitempty"`
	Status       string                 `json:"status"`
	Amount       int64                  `json:"amount"`
	Currency     string                 `json:"currency"`
	CreatedAt    time.Time              `json:"created_at"`
	ProviderData map[string]interface{} `json:"provider_data,omitempty"`
}

// CreateProductWithProvider creates a new one-time payment with a specific provider
func (u *ProductUseCase) CreateProductWithProvider(ctx context.Context, req *CreateProductRequest, paymentProvider provider.PaymentProvider) (*CreateProductResponse, error) {
	u.logger.Info("Creating one-time payment",
		zap.String("universal_id", req.UniversalID),
		zap.Int64("amount", req.Amount),
		zap.String("currency", req.Currency))

	// Generate order ID
	orderID := u.generateOrderID()

	// Initialize payment with provider
	providerReq := &provider.InitializePaymentRequest{
		UniversalID: req.UniversalID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		OrderID:     orderID,
		OrderName:   req.OrderName,
		CustomerKey: req.CustomerKey,
		PlanID:      req.PlanID,
		Metadata:    req.Metadata,
	}

	providerResp, err := paymentProvider.InitializePayment(ctx, providerReq)
	if err != nil {
		u.logger.Error("Failed to initialize payment with provider",
			zap.String("order_id", orderID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to initialize payment: %w", err)
	}

	// Create payment record in database
	payment := &entity.Payment{
		UniversalID:   req.UniversalID,
		TransactionID: orderID, // Store order ID in TransactionID field
		Amount:        float64(req.Amount),
		Currency:      req.Currency,
		Status:        entity.PaymentStatusPending,
		Method:        entity.PaymentMethodCard, // Default, will be updated on confirmation
		Description:   req.OrderName,
		Metadata:      providerResp.ProviderData,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			payment.Metadata[k] = v
		}
	}

	err = u.paymentRepo.CreateOneTimePayment(ctx, payment)
	if err != nil {
		u.logger.Error("Failed to create payment record",
			zap.String("order_id", orderID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	return &CreateProductResponse{
		OrderID:      orderID,
		PaymentKey:   providerResp.PaymentKey,
		ClientSecret: providerResp.ClientSecret,
		Status:       providerResp.Status,
		Amount:       providerResp.Amount,
		Currency:     providerResp.Currency,
		CreatedAt:    payment.CreatedAt,
		ProviderData: providerResp.ProviderData,
	}, nil
}

// ConfirmPaymentRequest represents a request to confirm a payment
type ConfirmProductRequest struct {
	OrderID    string                 `json:"order_id"`
	PaymentKey string                 `json:"payment_key"`
	Amount     int64                  `json:"amount"`
	Provider   string                 `json:"provider,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ConfirmPaymentResponse represents the response from payment confirmation
type ConfirmProductResponse struct {
	OrderID        string                 `json:"order_id"`
	PaymentKey     string                 `json:"payment_key"`
	TransactionKey string                 `json:"transaction_key,omitempty"`
	Status         string                 `json:"status"`
	Amount         int64                  `json:"amount"`
	Currency       string                 `json:"currency"`
	PaymentMethod  string                 `json:"payment_method,omitempty"`
	PaidAt         *time.Time             `json:"paid_at,omitempty"`
	ProviderData   map[string]interface{} `json:"provider_data,omitempty"`
}

// ConfirmPaymentWithProvider confirms a payment with a specific provider
func (u *ProductUseCase) ConfirmPaymentWithProvider(ctx context.Context, req *ConfirmProductRequest, paymentProvider provider.PaymentProvider) (*ConfirmProductResponse, error) {
	u.logger.Info("Confirming payment",
		zap.String("order_id", req.OrderID),
		zap.String("payment_key", req.PaymentKey),
		zap.Int64("amount", req.Amount))

	// Get existing payment record
	payment, err := u.paymentRepo.GetByOrderID(ctx, req.OrderID)
	if err != nil {
		u.logger.Error("Failed to get payment by order ID",
			zap.String("order_id", req.OrderID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment == nil {
		return nil, fmt.Errorf("payment not found for order ID: %s", req.OrderID)
	}

	// Validate amount matches
	if int64(payment.Amount) != req.Amount {
		u.logger.Error("Payment amount mismatch",
			zap.String("order_id", req.OrderID),
			zap.Int64("expected", int64(payment.Amount)),
			zap.Int64("received", req.Amount))
		return nil, fmt.Errorf("payment amount mismatch")
	}

	// Confirm with provider
	providerReq := &provider.ConfirmPaymentRequest{
		OrderID:      req.OrderID,
		PaymentKey:   req.PaymentKey,
		Amount:       req.Amount,
		ProviderData: req.Metadata,
	}

	providerResp, err := paymentProvider.ConfirmPayment(ctx, providerReq)
	if err != nil {
		u.logger.Error("Failed to confirm payment with provider",
			zap.String("order_id", req.OrderID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to confirm payment: %w", err)
	}

	// Update payment record
	updates := map[string]interface{}{
		"status":                     string(providerResp.Status),
		"provider_payment_intent_id": providerResp.PaymentKey,
		"provider_charge_id":         providerResp.TransactionKey,
		"payment_method_type":        providerResp.PaymentMethod,
		"provider_payment_data":      providerResp.ProviderData,
	}

	if providerResp.PaidAt != nil {
		updates["paid_at"] = providerResp.PaidAt
	}

	err = u.paymentRepo.UpdatePaymentAfterConfirm(ctx, req.OrderID, updates)
	if err != nil {
		u.logger.Error("Failed to update payment after confirmation",
			zap.String("order_id", req.OrderID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	return &ConfirmProductResponse{
		OrderID:        providerResp.OrderID,
		PaymentKey:     providerResp.PaymentKey,
		TransactionKey: providerResp.TransactionKey,
		Status:         string(providerResp.Status),
		Amount:         providerResp.Amount,
		Currency:       providerResp.Currency,
		PaymentMethod:  providerResp.PaymentMethod,
		PaidAt:         providerResp.PaidAt,
		ProviderData:   providerResp.ProviderData,
	}, nil
}

// generateOrderID generates a unique order ID
func (u *ProductUseCase) generateOrderID() string {
	return fmt.Sprintf("ORDER_%d_%s",
		time.Now().Unix(),
		uuid.New().String()[:8])
}

