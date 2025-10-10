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

// Subscription represents a subscription linked to a universal ID
type Subscription struct {
	ID                     int64              `gorm:"primaryKey;autoIncrement" json:"id"`
	UniversalID            uuid.UUID          `gorm:"column:universal_id;type:uuid;not null" json:"universal_id"`
	ProviderCustomerID       string             `gorm:"column:provider_customer_id;not null;size:100" json:"provider_customer_id"`
	ProviderSubscriptionID   *string            `gorm:"column:provider_subscription_id;unique;size:100" json:"provider_subscription_id,omitempty"`
	PlanID                 *string            `gorm:"not null;size:100" json:"plan_id,omitempty"`
	Status                 SubscriptionStatus `gorm:"type:subscription_status;not null;default:'active'" json:"status"`
	CurrentPeriodStart     time.Time          `gorm:"not null" json:"current_period_start"`
	CurrentPeriodEnd       time.Time          `gorm:"not null" json:"current_period_end"`
	CanceledAt             *time.Time         `json:"canceled_at,omitempty"`
	ProductName            string             `gorm:"size:255" json:"product_name"`
	Amount                 int64              `json:"amount"`
	Currency               string             `gorm:"size:3" json:"currency"`
	Interval               string             `gorm:"size:20" json:"interval"`
	IntervalCount          int64              `json:"interval_count"`
	ProviderSubscriptionData JSONB              `gorm:"column:provider_subscription_data;type:jsonb" json:"provider_subscription_data,omitempty"`
	CreatedAt              time.Time          `gorm:"default:now()" json:"created_at"`
	UpdatedAt              time.Time          `gorm:"default:now()" json:"updated_at"`

	// Relations
	Plan *PaymentPlan `gorm:"foreignKey:PlanID;references:ProviderProductID" json:"plan,omitempty"`
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
