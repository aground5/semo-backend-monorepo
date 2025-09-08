package model

import (
	"time"

	"github.com/google/uuid"
)

// CustomerMapping maps Stripe customer IDs to universal IDs
type CustomerMapping struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderCustomerID string    `gorm:"column:provider_customer_id;unique;not null;size:100;index" json:"provider_customer_id"`
	UniversalID      uuid.UUID `gorm:"column:universal_id;type:uuid;not null;index" json:"universal_id"`
	CustomerEmail    string    `gorm:"size:255" json:"customer_email"`
	CreatedAt        time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt        time.Time `gorm:"default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (CustomerMapping) TableName() string {
	return "customer_mappings"
}
