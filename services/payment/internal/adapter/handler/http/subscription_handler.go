package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	domainErrors "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/middleware/auth"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

type SubscriptionHandler struct {
	logger               *zap.Logger
	subscriptionService  *usecase.SubscriptionService
}

func NewSubscriptionHandler(logger *zap.Logger, subscriptionService *usecase.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		logger:               logger,
		subscriptionService:  subscriptionService,
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
