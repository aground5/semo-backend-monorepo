package entity

import "time"

type Payment struct {
	ID            string
	UserID        string
	Amount        float64
	Currency      string
	Status        PaymentStatus
	Method        PaymentMethod
	TransactionID string
	Description   string
	Metadata      map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCanceled   PaymentStatus = "canceled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

type PaymentMethod string

const (
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodBank   PaymentMethod = "bank"
	PaymentMethodWallet PaymentMethod = "wallet"
	PaymentMethodCrypto PaymentMethod = "crypto"
)
