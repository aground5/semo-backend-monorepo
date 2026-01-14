package model

import (
	"time"

	"github.com/google/uuid"
)

type BillingKey struct {
	ID                  int64      `gorm:"primaryKey;autoIncrement"`
	UniversalID         uuid.UUID  `gorm:"column:universal_id;type:uuid;not null"`
	CustomerKey         string     `gorm:"column:customer_key;uniqueIndex;size:300;not null"`
	EncryptedBillingKey string     `gorm:"column:encrypted_billing_key;type:text;not null"`
	EncryptionIV        string     `gorm:"column:encryption_iv;type:text;not null"`
	CardLastFour        string     `gorm:"column:card_last_four;size:4"`
	CardCompany         string     `gorm:"column:card_company;size:50"`
	CardType            string     `gorm:"column:card_type;size:20"`
	IsActive            bool       `gorm:"column:is_active;default:true"`
	CreatedAt           time.Time  `gorm:"default:now()"`
	UpdatedAt           time.Time  `gorm:"default:now()"`
	DeactivatedAt       *time.Time `gorm:"column:deactivated_at"`
}

func (BillingKey) TableName() string {
	return "billing_keys"
}

type BillingKeyAccessLog struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	BillingKeyID int64     `gorm:"column:billing_key_id;not null"`
	AccessType   string    `gorm:"column:access_type;size:50;not null"`
	AccessorID   string    `gorm:"column:accessor_id;size:100"`
	IPAddress    string    `gorm:"column:ip_address;size:45"`
	UserAgent    string    `gorm:"column:user_agent;type:text"`
	Purpose      string    `gorm:"column:purpose;size:255"`
	CreatedAt    time.Time `gorm:"default:now()"`
}

func (BillingKeyAccessLog) TableName() string {
	return "billing_key_access_logs"
}

type ScheduledPayment struct {
	ID             int64      `gorm:"primaryKey;autoIncrement"`
	SubscriptionID int64      `gorm:"column:subscription_id;not null"`
	BillingKeyID   int64      `gorm:"column:billing_key_id;not null"`
	ScheduledAt    time.Time  `gorm:"column:scheduled_at;not null"`
	Amount         int64      `gorm:"column:amount;not null"`
	Currency       string     `gorm:"column:currency;size:3;default:'KRW'"`
	OrderName      string     `gorm:"column:order_name;size:255"`
	Status         string     `gorm:"column:status;size:20;default:'pending'"`
	AttemptCount   int        `gorm:"column:attempt_count;default:0"`
	LastAttemptAt  *time.Time `gorm:"column:last_attempt_at"`
	LastError      string     `gorm:"column:last_error;type:text"`
	NextRetryAt    *time.Time `gorm:"column:next_retry_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
	PaymentID      *int64     `gorm:"column:payment_id"`
	CreatedAt      time.Time  `gorm:"default:now()"`
	UpdatedAt      time.Time  `gorm:"default:now()"`
}

func (ScheduledPayment) TableName() string {
	return "scheduled_payments"
}
