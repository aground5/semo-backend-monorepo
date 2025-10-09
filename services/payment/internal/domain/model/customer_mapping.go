package model

import (
	"time"

	"github.com/google/uuid"
)

// CustomerMapping maps provider customer IDs to universal IDs
type CustomerMapping struct {
	ID                 int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Provider           string    `gorm:"column:provider;size:50;not null;index:idx_customer_provider,priority:1;index:idx_universal_provider,priority:1" json:"provider"`
	ProviderCustomerID string    `gorm:"column:provider_customer_id;not null;size:100;index:idx_customer_provider,priority:2" json:"provider_customer_id"`
	UniversalID        uuid.UUID `gorm:"column:universal_id;type:uuid;not null;index:idx_universal_provider,priority:2" json:"universal_id"`
	CustomerEmail      string    `gorm:"size:255" json:"customer_email"`
	CreatedAt          time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt          time.Time `gorm:"default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (CustomerMapping) TableName() string {
	return "customer_mappings"
}
