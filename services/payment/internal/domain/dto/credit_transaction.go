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