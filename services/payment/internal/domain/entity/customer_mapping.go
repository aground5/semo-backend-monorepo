package entity

import "time"

// CustomerMapping stores the relationship between Stripe customer ID and user ID
type CustomerMapping struct {
	ID               int64     `json:"id"`
	StripeCustomerID string    `json:"stripe_customer_id"`
	UserID           string    `json:"user_id"` // Keep as string for compatibility with existing code
	Email            string    `json:"email"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
