package model

import (
	"database/sql/driver"
	"time"
)

// WebhookStatus represents the processing status of a webhook
type WebhookStatus string

const (
	WebhookStatusPending    WebhookStatus = "pending"
	WebhookStatusProcessing WebhookStatus = "processing"
	WebhookStatusCompleted  WebhookStatus = "completed"
	WebhookStatusFailed     WebhookStatus = "failed"
)

// Scan implements sql.Scanner interface
func (w *WebhookStatus) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		*w = WebhookStatus(v)
	case []byte:
		*w = WebhookStatus(v)
	default:
		*w = WebhookStatusPending
	}
	return nil
}

// Value implements driver.Valuer interface
func (w WebhookStatus) Value() (driver.Value, error) {
	return string(w), nil
}

// StripeWebhookEvent represents a Stripe webhook event
type StripeWebhookEvent struct {
	ID                 int64         `gorm:"primaryKey;autoIncrement" json:"id"`
	StripeEventID      string        `gorm:"unique;not null;size:255;index" json:"stripe_event_id"`
	EventType          string        `gorm:"not null;size:100;index" json:"event_type"`
	Status             WebhookStatus `gorm:"type:webhook_status;default:'pending';index" json:"status"`
	ProcessedAt        *time.Time    `json:"processed_at,omitempty"`
	Data               JSONB         `gorm:"type:jsonb;not null" json:"data"`
	APIVersion         *string       `gorm:"size:20" json:"api_version,omitempty"`
	ProcessingAttempts int           `gorm:"default:0" json:"processing_attempts"`
	LastError          *string       `json:"last_error,omitempty"`
	NextRetryAt        *time.Time    `json:"next_retry_at,omitempty"`
	CreatedAt          time.Time     `gorm:"default:now()" json:"created_at"`
	StripeCreatedAt    *time.Time    `json:"stripe_created_at,omitempty"`
}

// TableName specifies the table name for GORM
func (StripeWebhookEvent) TableName() string {
	return "stripe_webhook_events"
}
