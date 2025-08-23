package entity

import "time"

type Payment struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	Amount        float64                `json:"amount"`
	Currency      string                 `json:"currency"`
	Status        PaymentStatus          `json:"status"`
	Method        PaymentMethod          `json:"method"`
	TransactionID string                 `json:"transaction_id"`
	Description   string                 `json:"description"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
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
