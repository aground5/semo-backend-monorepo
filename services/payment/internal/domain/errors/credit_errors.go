package errors

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// InsufficientBalanceError is returned when a user doesn't have enough credits
type InsufficientBalanceError struct {
	Requested decimal.Decimal
	Available decimal.Decimal
}

func (e *InsufficientBalanceError) Error() string {
	return fmt.Sprintf("insufficient credit balance: requested %s, available %s", e.Requested.String(), e.Available.String())
}

// NewInsufficientBalanceError creates a new InsufficientBalanceError
func NewInsufficientBalanceError(requested, available decimal.Decimal) *InsufficientBalanceError {
	return &InsufficientBalanceError{
		Requested: requested,
		Available: available,
	}
}