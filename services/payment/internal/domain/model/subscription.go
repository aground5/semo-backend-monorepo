package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
)

// Scan implements sql.Scanner interface
func (s *SubscriptionStatus) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		*s = SubscriptionStatus(v)
	case []byte:
		*s = SubscriptionStatus(v)
	default:
		*s = SubscriptionStatusInactive
	}
	return nil
}

// Value implements driver.Valuer interface
func (s SubscriptionStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// Subscription represents a user's subscription
type Subscription struct {
	ID                     int64              `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID                 uuid.UUID          `gorm:"type:uuid;not null" json:"user_id"`
	StripeCustomerID       string             `gorm:"not null;size:100" json:"stripe_customer_id"`
	StripeSubscriptionID   *string            `gorm:"unique;size:100" json:"stripe_subscription_id,omitempty"`
	PlanID                 *int64             `gorm:"index" json:"plan_id,omitempty"`
	Status                 SubscriptionStatus `gorm:"type:subscription_status;not null;default:'active'" json:"status"`
	CurrentPeriodStart     time.Time          `gorm:"not null" json:"current_period_start"`
	CurrentPeriodEnd       time.Time          `gorm:"not null" json:"current_period_end"`
	CanceledAt             *time.Time         `json:"canceled_at,omitempty"`
	StripeSubscriptionData JSONB              `gorm:"type:jsonb" json:"stripe_subscription_data,omitempty"`
	CreatedAt              time.Time          `gorm:"default:now()" json:"created_at"`
	UpdatedAt              time.Time          `gorm:"default:now()" json:"updated_at"`

	// Relations
	Plan *SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

// JSONB represents a JSONB database type
type JSONB map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(src interface{}) error {
	if src == nil {
		*j = nil
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		*j = make(JSONB)
		return nil
	}
}

// TableName specifies the table name for GORM
func (Subscription) TableName() string {
	return "subscriptions"
}
