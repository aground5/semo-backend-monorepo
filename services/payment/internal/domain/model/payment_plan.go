package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Plan type constants
const (
	PlanTypeSubscription = "subscription"
	PlanTypeOneTime      = "one_time"
)

// PaymentPlan represents a payment plan (subscription or one-time)
type PaymentPlan struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderPriceID   string    `gorm:"column:provider_price_id;unique;not null;size:100" json:"provider_price_id"`
	ProviderProductID string    `gorm:"column:provider_product_id;not null;size:100" json:"provider_product_id"`
	PgProvider       string    `gorm:"column:pg_provider;size:50" json:"pg_provider"`
	DisplayName     string    `gorm:"not null;size:200" json:"display_name"`
	Type            string    `gorm:"not null;size:20;default:'subscription'" json:"type"` // 'subscription' or 'one_time'
	CreditsPerCycle int       `gorm:"not null" json:"credits_per_cycle"`
	Features        Features  `gorm:"type:jsonb;default:'{}'" json:"features"`
	SortOrder       int       `gorm:"default:0" json:"sort_order"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt       time.Time `gorm:"default:now()" json:"updated_at"`
}

// Features represents plan features as JSONB
type Features map[string]interface{}

// Value implements driver.Valuer interface
func (f Features) Value() (driver.Value, error) {
	return json.Marshal(f)
}

// Scan implements sql.Scanner interface
func (f *Features) Scan(src interface{}) error {
	if src == nil {
		*f = make(Features)
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, f)
	case string:
		return json.Unmarshal([]byte(v), f)
	default:
		*f = make(Features)
		return nil
	}
}

// TableName specifies the table name for GORM
func (PaymentPlan) TableName() string {
	return "payment_plans"
}
