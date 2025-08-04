package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreditTransactionDTO represents a simplified credit transaction for API responses
type CreditTransactionDTO struct {
	TransactionType string    `json:"transaction_type"`
	Amount          string    `json:"amount"`
	BalanceAfter    string    `json:"balance_after"`
	Description     string    `json:"description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// TransactionListResponse represents the paginated transaction list response
type TransactionListResponse struct {
	Transactions []CreditTransactionDTO `json:"transactions"`
	Pagination   PaginationInfo         `json:"pagination"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Total   int64 `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	HasMore bool  `json:"has_more"`
}

// TransactionFilters contains query filters for transaction retrieval
type TransactionFilters struct {
	UserID          uuid.UUID
	Limit           int
	Offset          int
	StartDate       *time.Time
	EndDate         *time.Time
	TransactionType *string
}

// SetDefaults sets default values for pagination
func (f *TransactionFilters) SetDefaults() {
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}

// UseCreditRequest represents the request body for using credits
type UseCreditRequest struct {
	Amount         string                 `json:"amount" validate:"required"`
	FeatureName    string                 `json:"feature_name" validate:"required,min=1,max=100"`
	Description    string                 `json:"description" validate:"required,min=1,max=500"`
	UsageMetadata  map[string]interface{} `json:"usage_metadata,omitempty"`
	IdempotencyKey *string                `json:"idempotency_key,omitempty" validate:"omitempty,uuid4"`
}

// UseCreditResponse represents the response for credit usage
type UseCreditResponse struct {
	Success       bool   `json:"success"`
	TransactionID int64  `json:"transaction_id"`
	BalanceAfter  string `json:"balance_after"`
	Message       string `json:"message"`
}