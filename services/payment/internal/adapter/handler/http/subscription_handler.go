package http

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"go.uber.org/zap"
)

type SubscriptionHandler struct {
	logger *zap.Logger
}

func NewSubscriptionHandler(logger *zap.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{logger: logger}
}

type SubscriptionStatus struct {
	Active       bool                 `json:"active"`
	Subscription *entity.Subscription `json:"subscription,omitempty"`
}

func (h *SubscriptionHandler) GetCurrentSubscription(c echo.Context) error {
	customerID := c.Request().Header.Get("X-Customer-ID")

	if customerID == "" {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error": "Customer ID required",
		})
	}

	h.logger.Info("Getting current subscription",
		zap.String("customer_id", customerID),
	)

	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String("all"),
	}
	params.AddExpand("data.items.data.price.product")

	iter := subscription.List(params)

	var activeSub *stripe.Subscription
	for iter.Next() {
		sub := iter.Subscription()
		if sub.Status == "active" || sub.Status == "trialing" {
			activeSub = sub
			break
		}
	}

	if err := iter.Err(); err != nil {
		h.logger.Error("Error listing subscriptions", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	if activeSub == nil {
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "No active subscription found",
		})
	}

	var items []entity.SubscriptionItem
	for _, item := range activeSub.Items.Data {
		var productName string
		if item.Price.Product != nil {
			productName = item.Price.Product.Name
		}

		var interval string
		var intervalCount int64
		if item.Price.Recurring != nil {
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

	return c.JSON(http.StatusOK, entity.Subscription{
		ID:                activeSub.ID,
		CustomerID:        customerID,
		Status:            string(activeSub.Status),
		CurrentPeriodEnd:  time.Unix(activeSub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd: activeSub.CancelAtPeriodEnd,
		Items:             items,
	})
}

func (h *SubscriptionHandler) CancelSubscription(c echo.Context) error {
	subscriptionID := c.Param("id")

	h.logger.Info("Canceling subscription",
		zap.String("subscription_id", subscriptionID),
	)

	sub, err := subscription.Update(
		subscriptionID,
		&stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		},
	)

	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error": "Subscription not found",
			})
		}

		h.logger.Error("Error canceling subscription", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	h.logger.Info("Subscription canceled",
		zap.String("subscription_id", sub.ID),
		zap.Time("cancel_at", time.Unix(sub.CurrentPeriodEnd, 0)),
	)

	return c.JSON(http.StatusOK, echo.Map{
		"id":                   sub.ID,
		"status":               sub.Status,
		"cancel_at_period_end": sub.CancelAtPeriodEnd,
		"cancel_at":            sub.CurrentPeriodEnd,
		"message":              "Subscription will be canceled at the end of the current period",
	})
}
