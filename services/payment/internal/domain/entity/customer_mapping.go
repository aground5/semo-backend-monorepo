package entity

import "time"

// CustomerMapping stores the relationship between Stripe customer ID and universal ID
type CustomerMapping struct {
	ID               int64     `json:"id"`
	StripeCustomerID string    `json:"provider_customer_id"`
	UniversalID      string    `json:"universal_id"` // Keep as string for compatibility with existing code
	Email            string    `json:"email"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
