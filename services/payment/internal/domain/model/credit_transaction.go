package model

import (
	"database/sql/driver"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransactionType represents the type of credit transaction
type TransactionType string

const (
	TransactionTypeCreditAllocation TransactionType = "credit_allocation"
	TransactionTypeCreditUsage      TransactionType = "credit_usage"
	TransactionTypeRefund           TransactionType = "refund"
	TransactionTypeAdjustment       TransactionType = "adjustment"
)

// Scan implements sql.Scanner interface
func (t *TransactionType) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		*t = TransactionType(v)
	case []byte:
		*t = TransactionType(v)
	default:
		return nil
	}
	return nil
}

// Value implements driver.Valuer interface
func (t TransactionType) Value() (driver.Value, error) {
	return string(t), nil
}

// CreditTransaction represents a credit transaction
type CreditTransaction struct {
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uuid.UUID       `gorm:"type:uuid;not null;index:idx_credit_transactions_user_created" json:"user_id"`
	SubscriptionID  *int64          `gorm:"index" json:"subscription_id,omitempty"`
	TransactionType TransactionType `gorm:"type:transaction_type;not null" json:"transaction_type"`
	Amount          decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	BalanceAfter    decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"balance_after"`
	Description     string          `gorm:"not null" json:"description"`
	FeatureName     *string         `gorm:"size:100" json:"feature_name,omitempty"`
	UsageMetadata   JSONB           `gorm:"type:jsonb;default:'{}'" json:"usage_metadata"`
	ReferenceID     *string         `gorm:"size:200;index:idx_credit_transactions_reference,where:reference_id IS NOT NULL" json:"reference_id,omitempty"`
	IdempotencyKey  *uuid.UUID      `gorm:"type:uuid;unique" json:"idempotency_key,omitempty"`
	CreatedAt       time.Time       `gorm:"default:now();index:idx_credit_transactions_user_created" json:"created_at"`

	// Relations
	Subscription *Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
}

// TableName specifies the table name for GORM
func (CreditTransaction) TableName() string {
	return "credit_transactions"
}

// UserCreditBalance represents the current credit balance for a user
type UserCreditBalance struct {
	UserID            uuid.UUID       `gorm:"type:uuid;primaryKey" json:"user_id"`
	CurrentBalance    decimal.Decimal `gorm:"type:decimal(15,2)" json:"current_balance"`
	LastTransactionAt time.Time       `json:"last_transaction_at"`
}

// TableName specifies the table name for GORM
func (UserCreditBalance) TableName() string {
	return "user_credit_balances"
}
