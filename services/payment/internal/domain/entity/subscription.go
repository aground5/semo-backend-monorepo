package entity

import "time"

type Subscription struct {
	ID                string             `json:"id"`
	CustomerID        string             `json:"customer_id"`
	CustomerEmail     string             `json:"customer_email"`
	Status            string             `json:"status"`
	CurrentPeriodEnd  time.Time          `json:"current_period_end"`
	CancelAtPeriodEnd bool               `json:"cancel_at_period_end"`
	Items             []SubscriptionItem `json:"items"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type SubscriptionItem struct {
	ProductName   string `json:"product_name"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Interval      string `json:"interval"`
	IntervalCount int64  `json:"interval_count"`
}

type Plan struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Type          string `json:"type"` // 'subscription' or 'one_time'
	Interval      string `json:"interval,omitempty"`
	IntervalCount int64  `json:"interval_count,omitempty"`
}

type WebhookData struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	CustomerID     string    `json:"customer_id"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	InvoiceID      string    `json:"invoice_id,omitempty"`
	Amount         int64     `json:"amount,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}
