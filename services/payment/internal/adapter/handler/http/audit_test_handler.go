package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

// AuditTestHandler is for testing audit logging functionality
type AuditTestHandler struct {
	logger      *zap.Logger
	paymentRepo repository.PaymentRepository
}

// NewAuditTestHandler creates a new audit test handler
func NewAuditTestHandler(logger *zap.Logger, paymentRepo repository.PaymentRepository) *AuditTestHandler {
	return &AuditTestHandler{
		logger:      logger,
		paymentRepo: paymentRepo,
	}
}

// TestAuditLog creates a test payment record to verify audit logging
func (h *AuditTestHandler) TestAuditLog(c echo.Context) error {
	// Generate test data
	testUserID := uuid.New().String()
	testPayment := &entity.Payment{
		UserID:        testUserID,
		Amount:        1000.00,
		Currency:      "USD",
		Status:        entity.PaymentStatusCompleted,
		Method:        entity.PaymentMethodCard,
		TransactionID: "test_" + uuid.New().String(),
		Description:   "Test payment for audit logging",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	h.logger.Info("Creating test payment for audit logging",
		zap.String("user_id", testUserID),
		zap.Float64("amount", testPayment.Amount))

	// Create payment (should trigger audit log)
	if err := h.paymentRepo.Create(c.Request().Context(), testPayment); err != nil {
		h.logger.Error("Failed to create test payment", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Failed to create test payment",
			"details": err.Error(),
		})
	}

	h.logger.Info("Test payment created successfully",
		zap.String("payment_id", testPayment.ID))

	// Now update the payment (should trigger another audit log)
	testPayment.Status = entity.PaymentStatusRefunded
	testPayment.Description = "Updated: Test payment refunded"

	if err := h.paymentRepo.Update(c.Request().Context(), testPayment); err != nil {
		h.logger.Error("Failed to update test payment", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Failed to update test payment",
			"details": err.Error(),
		})
	}

	h.logger.Info("Test payment updated successfully")

	return c.JSON(http.StatusOK, echo.Map{
		"success":    true,
		"message":    "Audit log test completed. Check audit_log table for entries.",
		"payment_id": testPayment.ID,
		"user_id":    testUserID,
		"operations": []string{
			"INSERT - Created new payment",
			"UPDATE - Changed status to refunded",
		},
		"check_query": "SELECT * FROM audit_log WHERE table_name = 'payments' ORDER BY created_at DESC LIMIT 10;",
	})
}
