package model

import (
	"time"

	"github.com/google/uuid"
)

// CustomerMapping maps Stripe customer IDs to user IDs
type CustomerMapping struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	StripeCustomerID string    `gorm:"unique;not null;size:100;index" json:"stripe_customer_id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	CustomerEmail    string    `gorm:"size:255" json:"customer_email"`
	CreatedAt        time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt        time.Time `gorm:"default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (CustomerMapping) TableName() string {
	return "customer_mappings"
}
