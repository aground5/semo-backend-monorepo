package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Payment represents a payment record
type Payment struct {
	ID                    int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UniversalID           uuid.UUID       `gorm:"column:universal_id;type:uuid;not null;index" json:"universal_id"`
	SubscriptionID        *int64          `gorm:"index" json:"subscription_id,omitempty"`
	ProviderPaymentIntentID *string         `gorm:"column:provider_payment_intent_id;unique;size:100" json:"provider_payment_intent_id,omitempty"`
	ProviderChargeID        *string         `gorm:"column:provider_charge_id;size:100" json:"provider_charge_id,omitempty"`
	ProviderInvoiceID       *string         `gorm:"column:provider_invoice_id;size:100" json:"provider_invoice_id,omitempty"`
	AmountCents           int             `gorm:"not null" json:"amount_cents"`
	Currency              string          `gorm:"size:3;default:'KRW'" json:"currency"`
	Status                string          `gorm:"size:50;not null" json:"status"`
	CreditsAllocated      decimal.Decimal `gorm:"type:decimal(15,2);default:0" json:"credits_allocated"`
	CreditsAllocatedAt    *time.Time      `json:"credits_allocated_at,omitempty"`
	PaymentMethodType     *string         `gorm:"size:50" json:"payment_method_type,omitempty"`
	FailureCode           *string         `gorm:"size:100" json:"failure_code,omitempty"`
	FailureMessage        *string         `json:"failure_message,omitempty"`
	ProviderPaymentData     JSONB           `gorm:"column:provider_payment_data;type:jsonb" json:"provider_payment_data,omitempty"`
	PaidAt                *time.Time      `json:"paid_at,omitempty"`
	CreatedAt             time.Time       `gorm:"default:now()" json:"created_at"`
	UpdatedAt             time.Time       `gorm:"default:now()" json:"updated_at"`

	// Relations
	Subscription *Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
}

// TableName specifies the table name for GORM
func (Payment) TableName() string {
	return "payments"
}
