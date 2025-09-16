package model

import (
	"time"
)

// TossWebhookEvent represents a TossPayments webhook event
type TossWebhookEvent struct {
	ID                 int64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TossEventID        string        `gorm:"unique;size:255;index" json:"toss_event_id"`
	EventType          string        `gorm:"not null;size:100;index" json:"event_type"`
	EventStatus        *string       `gorm:"size:50" json:"event_status,omitempty"`
	ProcessingStatus   WebhookStatus `gorm:"type:webhook_status;default:'pending';index" json:"processing_status"`
	ProcessedAt        *time.Time    `json:"processed_at,omitempty"`
	PaymentKey         *string       `gorm:"size:200" json:"payment_key,omitempty"`
	OrderID            *string       `gorm:"size:200;index" json:"order_id,omitempty"`
	TransactionKey     *string       `gorm:"size:200" json:"transaction_key,omitempty"`
	Secret             *string       `gorm:"size:200" json:"secret,omitempty"`
	EventData          JSONB         `gorm:"type:jsonb;not null" json:"event_data"`
	RetryCount         int           `gorm:"default:0" json:"retry_count"`
	LastError          *string       `json:"last_error,omitempty"`
	NextRetryAt        *time.Time    `json:"next_retry_at,omitempty"`
	IPAddress          *string       `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent          *string       `json:"user_agent,omitempty"`
	TossCreatedAt      time.Time     `gorm:"not null" json:"toss_created_at"`
	CreatedAt          time.Time     `gorm:"default:now()" json:"created_at"`
	UpdatedAt          time.Time     `gorm:"default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (TossWebhookEvent) TableName() string {
	return "toss_webhook_events"
}