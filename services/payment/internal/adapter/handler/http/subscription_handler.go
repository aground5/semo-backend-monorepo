package http

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/subscription"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	domainErrors "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/middleware/auth"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

type SubscriptionHandler struct {
	logger               *zap.Logger
	subscriptionService  *usecase.SubscriptionService
	customerMappingRepo  domainRepo.CustomerMappingRepository  // 추가
	clientURL            string                               // 추가
}

func NewSubscriptionHandler(
	logger *zap.Logger, 
	subscriptionService *usecase.SubscriptionService,
	customerMappingRepo domainRepo.CustomerMappingRepository,  // 추가
	clientURL string,                                          // 추가
) *SubscriptionHandler {
	return &SubscriptionHandler{
		logger:               logger,
		subscriptionService:  subscriptionService,
		customerMappingRepo:  customerMappingRepo,              // 추가
		clientURL:            clientURL,                        // 추가
	}
}

func (h *SubscriptionHandler) GetCurrentSubscription(c echo.Context) error {
	// Get authenticated user from JWT
	user, err := auth.RequireAuth(c)
	if err != nil {
		return err // RequireAuth already returns the JSON error response
	}

	h.logger.Info("Getting current subscription",
		zap.String("universal_id", user.UniversalID),
	)

	// Get active subscription for the user
	activeSub, err := h.subscriptionService.GetActiveSubscriptionForUniversalID(c.Request().Context(), user.UniversalID)
	if err != nil {
		if errors.Is(err, domainErrors.ErrNoCustomerMapping) {
			// Customer mapping doesn't exist - create it lazily
			h.logger.Info("No customer mapping found, creating one",
				zap.String("universal_id", user.UniversalID))
			
			// Create customer mapping with user's email
			if err := h.getOrCreateCustomerMapping(c, user); err != nil {
				h.logger.Error("Failed to create customer mapping",
					zap.String("universal_id", user.UniversalID),
					zap.Error(err))
				return c.JSON(http.StatusInternalServerError, echo.Map{
					"error": "Failed to initialize customer account",
				})
			}
			
			// Return HTTP 204 No Content (no active subscription)
			return c.NoContent(http.StatusNoContent)
		}
		if errors.Is(err, domainErrors.ErrNoActiveSubscription) {
			// Customer exists but no active subscription
			return c.NoContent(http.StatusNoContent)
		}
		// Other errors
		h.logger.Error("Failed to get active subscription",
			zap.String("universal_id", user.UniversalID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to retrieve subscription information",
		})
	}

	customerID := ""
	if activeSub.Customer != nil {
		customerID = activeSub.Customer.ID
	}

	// Extract subscription product information
	// Since we now support only one subscription per customer, we take the first item
	var productName string
	var amount int64
	var currency string
	var interval string
	var intervalCount int64

	if len(activeSub.Items.Data) > 0 {
		item := activeSub.Items.Data[0]
		if item.Price != nil {
			// First try: Price nickname (often contains the product name)
			if item.Price.Nickname != "" {
				productName = item.Price.Nickname
			} else if item.Price.Product != nil && item.Price.Product.Name != "" {
				// Second try: Product name if already expanded
				productName = item.Price.Product.Name
			} else {
				// Fallback: Use a generic name
				productName = "Subscription"
			}

			amount = item.Price.UnitAmount
			currency = string(item.Price.Currency)

			if item.Price.Recurring != nil {
				interval = string(item.Price.Recurring.Interval)
				intervalCount = item.Price.Recurring.IntervalCount
			}
		}
	}

	var planID *string

	if len(activeSub.Items.Data) > 0 {
		item := activeSub.Items.Data[0]
		if item.Price != nil && item.Price.Product != nil {
			id := item.Price.Product.ID
			planID = &id
		}
	}

	h.logger.Info("Active subscription found",
		zap.String("subscription_id", activeSub.ID),
		zap.String("universal_id", user.UniversalID),
		zap.String("status", string(activeSub.Status)),
	)

	return c.JSON(http.StatusOK, entity.Subscription{
		ID:                activeSub.ID,
		CustomerID:        customerID,
		Status:            string(activeSub.Status),
		CurrentPeriodEnd:  time.Unix(activeSub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd: activeSub.CancelAtPeriodEnd,
		ProductName:       productName,
		Amount:            amount,
		Currency:          currency,
		Interval:          interval,
		IntervalCount:     intervalCount,
		PlanID:            planID,
	})
}

// getOrCreateCustomerMapping ensures a Stripe customer exists for the authenticated user
func (h *SubscriptionHandler) getOrCreateCustomerMapping(c echo.Context, user *auth.AuthUser) error {
	// Check if we already have a Stripe customer for this user
	existingMapping, err := h.customerMappingRepo.GetByUniversalID(c.Request().Context(), user.UniversalID)
	if err == nil && existingMapping != nil {
		// Customer mapping already exists
		return nil
	}

	// Create new Stripe customer
	customerParams := &stripe.CustomerParams{
		Email: stripe.String(user.Email),
		Metadata: map[string]string{
			"universal_id": user.UniversalID,
		},
	}
	stripeCustomer, err := customer.New(customerParams)
	if err != nil {
		return fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	h.logger.Info("Created new Stripe customer for user",
		zap.String("customer_id", stripeCustomer.ID),
		zap.String("universal_id", user.UniversalID),
		zap.String("email", user.Email))

	// Save customer mapping
	mapping := &entity.CustomerMapping{
		UniversalID:      user.UniversalID,
		StripeCustomerID: stripeCustomer.ID,
		Email:            user.Email,
	}
	if err := h.customerMappingRepo.Create(c.Request().Context(), mapping); err != nil {
		return fmt.Errorf("failed to save customer mapping: %w", err)
	}

	return nil
}

func (h *SubscriptionHandler) CreateSubscription(c echo.Context) error {
	// Get authenticated user from JWT
	user, err := auth.RequireAuth(c)
	if err != nil {
		return err // RequireAuth already returns the JSON error response
	}

	var req CreateCheckoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request body",
		})
	}

	// Validate user ID from JWT is a valid UUID
	if _, err := uuid.Parse(user.UniversalID); err != nil {
		h.logger.Error("Invalid user ID format from JWT",
			zap.String("universal_id", user.UniversalID),
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Invalid user ID in authentication token",
			"details": "Expected format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		})
	}

	h.logger.Info("Creating subscription with Payment Element...",
		zap.String("price_id", req.PriceID),
		zap.String("email", req.Email),
		zap.String("universal_id", user.UniversalID),
		zap.String("jwt_email", user.Email),
	)

	// Check if we already have a Stripe customer for this user
	var customerID string
	if h.customerMappingRepo != nil {
		existingMapping, err := h.customerMappingRepo.GetByUniversalID(c.Request().Context(), user.UniversalID)
		if err != nil {
			h.logger.Warn("Error checking for existing customer mapping",
				zap.String("universal_id", user.UniversalID),
				zap.Error(err))
		} else if existingMapping != nil {
			customerID = existingMapping.StripeCustomerID
			h.logger.Info("Found existing Stripe customer",
				zap.String("customer_id", customerID),
				zap.String("universal_id", user.UniversalID))
		}
	}

	// Create or retrieve customer
	if customerID == "" {
		// Create new customer
		customerParams := &stripe.CustomerParams{
			Email: stripe.String(req.Email),
			Metadata: map[string]string{
				"universal_id": user.UniversalID,
			},
		}
		customer, err := customer.New(customerParams)
		if err != nil {
			h.logger.Error("Error creating customer", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "Failed to create customer",
			})
		}
		customerID = customer.ID
		h.logger.Info("Created new Stripe customer",
			zap.String("customer_id", customerID),
			zap.String("email", req.Email))

		// Save customer mapping if repository is available
		if h.customerMappingRepo != nil {
			// Parse user ID to UUID
			parsedUniversalID, err := uuid.Parse(user.UniversalID)
			if err != nil {
				h.logger.Error("Failed to parse user ID",
					zap.String("universal_id", user.UniversalID),
					zap.Error(err))
				return c.JSON(http.StatusInternalServerError, echo.Map{
					"error": "Invalid user ID format",
				})
			}

			mapping := &entity.CustomerMapping{
				UniversalID:      parsedUniversalID.String(),
				StripeCustomerID: customerID,
			}
			if err := h.customerMappingRepo.Create(c.Request().Context(), mapping); err != nil {
				h.logger.Warn("Failed to save customer mapping",
					zap.String("universal_id", user.UniversalID),
					zap.String("customer_id", customerID),
					zap.Error(err))
			}
		}
	}

	// Create subscription with payment_behavior set to default_incomplete
	// This creates the subscription in an incomplete state and returns a payment intent
	subscriptionParams := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(req.PriceID),
			},
		},
		PaymentBehavior: stripe.String("default_incomplete"),
		PaymentSettings: &stripe.SubscriptionPaymentSettingsParams{
			SaveDefaultPaymentMethod: stripe.String("on_subscription"),
			PaymentMethodTypes:       stripe.StringSlice([]string{"card"}),
		},
		Expand: stripe.StringSlice([]string{
			"latest_invoice.payment_intent",
			"pending_setup_intent",
		}),
		Metadata: map[string]string{
			"universal_id": user.UniversalID,
		},
	}

	// Create the subscription
	sub, err := subscription.New(subscriptionParams)
	if err != nil {
		h.logger.Error("Error creating subscription", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to create subscription",
		})
	}

	var clientSecret string
	var intentType string

	// Get the client secret from either payment intent or setup intent
	if sub.PendingSetupIntent != nil && sub.PendingSetupIntent.ClientSecret != "" {
		clientSecret = sub.PendingSetupIntent.ClientSecret
		intentType = "setup_intent"
		h.logger.Info("Using setup intent for subscription",
			zap.String("setup_intent_id", sub.PendingSetupIntent.ID))
	} else if sub.LatestInvoice != nil && sub.LatestInvoice.PaymentIntent != nil {
		clientSecret = sub.LatestInvoice.PaymentIntent.ClientSecret
		intentType = "payment_intent"
		h.logger.Info("Using payment intent for subscription",
			zap.String("payment_intent_id", sub.LatestInvoice.PaymentIntent.ID))
	} else {
		h.logger.Error("No payment intent or setup intent found in subscription")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to get payment intent",
		})
	}

	h.logger.Info("Subscription created successfully",
		zap.String("subscription_id", sub.ID),
		zap.String("customer_id", customerID),
		zap.String("intent_type", intentType),
		zap.String("status", string(sub.Status)))

	// Return the response with client secret for Payment Element
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"subscriptionId": sub.ID,
		"clientSecret":   clientSecret,
		"intentType":     intentType,
		"status":         string(sub.Status),
		"customerId":     customerID,
	})
}

// CancelCurrentSubscription cancels the authenticated user's active subscription
// This is a secure endpoint that uses JWT authentication to identify the user
// and automatically finds their active subscription to cancel
func (h *SubscriptionHandler) CancelCurrentSubscription(c echo.Context) error {
	// Get authenticated user from JWT
	user, err := auth.RequireAuth(c)
	if err != nil {
		return err // RequireAuth already returns the JSON error response
	}

	h.logger.Info("Attempting to cancel current subscription",
		zap.String("universal_id", user.UniversalID),
	)

	// Cancel the user's active subscription
	updatedSub, err := h.subscriptionService.CancelSubscriptionForUniversalID(c.Request().Context(), user.UniversalID)
	if err != nil {
		h.logger.Error("Failed to cancel subscription",
			zap.String("universal_id", user.UniversalID),
			zap.Error(err))
		
		// Handle specific error cases
		if errors.Is(err, domainErrors.ErrNoCustomerMapping) {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":   "No subscription found",
				"message": "User has no associated Stripe customer",
				"code":    "NO_CUSTOMER_MAPPING",
			})
		}
		if errors.Is(err, domainErrors.ErrNoActiveSubscription) {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":   "No active subscription found",
				"message": "User has no active subscription to cancel",
				"code":    "NO_ACTIVE_SUBSCRIPTION",
			})
		}
		
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to cancel subscription",
			"code":  "CANCELLATION_FAILED",
		})
	}

	h.logger.Info("Subscription canceled successfully",
		zap.String("subscription_id", updatedSub.ID),
		zap.String("universal_id", user.UniversalID),
		zap.Time("cancel_at", time.Unix(updatedSub.CurrentPeriodEnd, 0)),
	)

	// Build response with subscription details
	// Extract subscription product information - take the first item
	var productName string
	var amount int64
	var currency string
	var interval string
	var intervalCount int64

	if len(updatedSub.Items.Data) > 0 {
		item := updatedSub.Items.Data[0]
		if item.Price != nil {
			// First try: Price nickname (often contains the product name)
			if item.Price.Nickname != "" {
				productName = item.Price.Nickname
			} else if item.Price.Product != nil && item.Price.Product.Name != "" {
				// Second try: Product name if already expanded
				productName = item.Price.Product.Name
			} else {
				// Fallback: Use a generic name
				productName = "Subscription"
			}

			amount = item.Price.UnitAmount
			currency = string(item.Price.Currency)

			if item.Price.Recurring != nil {
				interval = string(item.Price.Recurring.Interval)
				intervalCount = item.Price.Recurring.IntervalCount
			}
		}
	}

	customerID := ""
	if updatedSub.Customer != nil {
		customerID = updatedSub.Customer.ID
	}

	return c.JSON(http.StatusOK, echo.Map{
		"subscription": entity.Subscription{
			ID:                updatedSub.ID,
			CustomerID:        customerID,
			Status:            string(updatedSub.Status),
			CurrentPeriodEnd:  time.Unix(updatedSub.CurrentPeriodEnd, 0),
			CancelAtPeriodEnd: updatedSub.CancelAtPeriodEnd,
			ProductName:       productName,
			Amount:            amount,
			Currency:          currency,
			Interval:          interval,
			IntervalCount:     intervalCount,
		},
		"message": "Subscription will be canceled at the end of the current billing period",
		"cancel_at": time.Unix(updatedSub.CurrentPeriodEnd, 0).Format(time.RFC3339),
	})
}
