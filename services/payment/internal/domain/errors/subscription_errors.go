package errors

import "errors"

var (
	// ErrNoCustomerMapping indicates that the user has no associated Stripe customer
	ErrNoCustomerMapping = errors.New("no customer mapping found for user")

	// ErrNoActiveSubscription indicates that the customer has no active subscription
	ErrNoActiveSubscription = errors.New("no active subscription found")

	// ErrSubscriptionNotFound indicates that the specified subscription was not found
	ErrSubscriptionNotFound = errors.New("subscription not found")

	// ErrCancellationFailed indicates that the subscription cancellation failed
	ErrCancellationFailed = errors.New("failed to cancel subscription")
)