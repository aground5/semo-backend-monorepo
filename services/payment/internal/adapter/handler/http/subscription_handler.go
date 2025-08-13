package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v79"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
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

type SubscriptionStatus struct {
	Active       bool                 `json:"active"`
	Subscription *entity.Subscription `json:"subscription,omitempty"`
}

func (h *SubscriptionHandler) GetCurrentSubscription(c echo.Context) error {
	// Get authenticated user from JWT
	user, err := auth.RequireAuth(c)
	if err != nil {
		return err // RequireAuth already returns the JSON error response
	}

	h.logger.Info("Getting current subscription",
		zap.String("user_id", user.UserID),
	)

	// Get active subscription for the user
	activeSub, err := h.subscriptionService.GetActiveSubscriptionForUser(c.Request().Context(), user.UserID)
	if err != nil {
		h.logger.Error("Failed to get active subscription",
			zap.String("user_id", user.UserID),
			zap.Error(err))
		if errors.Is(err, domainErrors.ErrNoCustomerMapping) {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":   "No subscription found",
				"message": "User has no associated Stripe customer",
			})
		}
		if errors.Is(err, domainErrors.ErrNoActiveSubscription) {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error": "No active subscription found",
			})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to retrieve subscription information",
		})
	}

	customerID := ""
	if activeSub.Customer != nil {
		customerID = activeSub.Customer.ID
	}

	var items []entity.SubscriptionItem
	for _, item := range activeSub.Items.Data {
		var productName string
		// Try to get product name from different sources
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
		}

		var interval string
		var intervalCount int64
		if item.Price != nil && item.Price.Recurring != nil {
			interval = string(item.Price.Recurring.Interval)
			intervalCount = item.Price.Recurring.IntervalCount
		}

		items = append(items, entity.SubscriptionItem{
			ProductName:   productName,
			Amount:        item.Price.UnitAmount,
			Currency:      string(item.Price.Currency),
			Interval:      interval,
			IntervalCount: intervalCount,
		})
	}

	h.logger.Info("Active subscription found",
		zap.String("subscription_id", activeSub.ID),
		zap.String("user_id", user.UserID),
		zap.String("status", string(activeSub.Status)),
	)

	return c.JSON(http.StatusOK, entity.Subscription{
		ID:                activeSub.ID,
		CustomerID:        customerID,
		Status:            string(activeSub.Status),
		CurrentPeriodEnd:  time.Unix(activeSub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd: activeSub.CancelAtPeriodEnd,
		Items:             items,
	})
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
	if _, err := uuid.Parse(user.UserID); err != nil {
		h.logger.Error("Invalid user ID format from JWT",
			zap.String("user_id", user.UserID),
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Invalid user ID in authentication token",
			"details": "Expected format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		})
	}

	h.logger.Info("Creating subscription...",
		zap.String("price_id", req.PriceID),
		zap.String("email", req.Email),
		zap.String("user_id", user.UserID),
		zap.String("jwt_email", user.Email),
		zap.String("mode", req.Mode),
	)

	// Check if we already have a Stripe customer for this user
	var existingCustomerID string
	if h.customerMappingRepo != nil {
		existingMapping, err := h.customerMappingRepo.GetByUserID(c.Request().Context(), user.UserID)
		if err != nil {
			h.logger.Warn("Error checking for existing customer mapping",
				zap.String("user_id", user.UserID),
				zap.Error(err))
		} else if existingMapping != nil {
			existingCustomerID = existingMapping.StripeCustomerID
			h.logger.Info("Found existing Stripe customer",
				zap.String("customer_id", existingCustomerID),
				zap.String("user_id", user.UserID))
		}
	}

	// 기본 파라미터 설정
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		// Set metadata on subscription
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": user.UserID,
			},
		},
		// Session metadata
		Metadata: map[string]string{
			"user_id": user.UserID,
		},
	}

	// Embedded Checkout 설정
	params.UIMode = stripe.String("embedded")
	params.ReturnURL = stripe.String(h.clientURL + "/?payment_complete=true&session_id={CHECKOUT_SESSION_ID}")
	h.logger.Info("Using embedded checkout mode")

	// Use existing customer or create new one
	if existingCustomerID != "" {
		params.Customer = stripe.String(existingCustomerID)
		h.logger.Info("Using existing customer for checkout session",
			zap.String("customer_id", existingCustomerID))
	} else {
		params.CustomerEmail = stripe.String(req.Email)
		h.logger.Info("Creating new customer with email",
			zap.String("email", req.Email))
	}

	s, err := checkoutsession.New(params)
	if err != nil {
		h.logger.Error("Error creating subscription", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	// Mode에 따라 다른 응답 반환
	h.logger.Info("Checkout session created for embedded mode",
		zap.String("session_id", s.ID),
		zap.Bool("has_client_secret", s.ClientSecret != ""))

	return c.JSON(http.StatusCreated, CreateCheckoutResponse{
		ID:           s.ID,
		ClientSecret: s.ClientSecret,
		Status:       "pending",
		SessionID:    s.ID,
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
		zap.String("user_id", user.UserID),
	)

	// Cancel the user's active subscription
	updatedSub, err := h.subscriptionService.CancelSubscriptionForUser(c.Request().Context(), user.UserID)
	if err != nil {
		h.logger.Error("Failed to cancel subscription",
			zap.String("user_id", user.UserID),
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
		zap.String("user_id", user.UserID),
		zap.Time("cancel_at", time.Unix(updatedSub.CurrentPeriodEnd, 0)),
	)

	// Build response with subscription details
	var items []entity.SubscriptionItem
	for _, item := range updatedSub.Items.Data {
		var productName string
		// Try to get product name from different sources
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
		}

		var interval string
		var intervalCount int64
		if item.Price != nil && item.Price.Recurring != nil {
			interval = string(item.Price.Recurring.Interval)
			intervalCount = item.Price.Recurring.IntervalCount
		}

		items = append(items, entity.SubscriptionItem{
			ProductName:   productName,
			Amount:        item.Price.UnitAmount,
			Currency:      string(item.Price.Currency),
			Interval:      interval,
			IntervalCount: intervalCount,
		})
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
			Items:             items,
		},
		"message": "Subscription will be canceled at the end of the current billing period",
		"cancel_at": time.Unix(updatedSub.CurrentPeriodEnd, 0).Format(time.RFC3339),
	})
}
