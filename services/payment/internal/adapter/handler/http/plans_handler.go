package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"go.uber.org/zap"
)

type PlansHandler struct {
	logger *zap.Logger
}

func NewPlansHandler(logger *zap.Logger) *PlansHandler {
	return &PlansHandler{logger: logger}
}

func (h *PlansHandler) GetPlans(c echo.Context) error {
	h.logger.Info("Fetching subscription plans...")
	
	params := &stripe.PriceListParams{
		Active: stripe.Bool(true),
		Type:   stripe.String(string(stripe.PriceTypeRecurring)),
	}
	params.AddExpand("data.product")
	params.Limit = stripe.Int64(100)
	
	iter := price.List(params)
	
	var plans []entity.Plan
	var totalCount int
	
	for iter.Next() {
		p := iter.Price()
		totalCount++
		
		if p.Product != nil && p.Product.Active {
			h.logger.Debug("Plan found",
				zap.String("price_id", p.ID),
				zap.String("product_name", p.Product.Name),
				zap.Int64("amount", p.UnitAmount),
				zap.String("currency", string(p.Currency)),
			)
			
			plan := entity.Plan{
				ID:            p.ID,
				Name:          p.Product.Name,
				Description:   p.Product.Description,
				Amount:        p.UnitAmount,
				Currency:      string(p.Currency),
				Interval:      string(p.Recurring.Interval),
				IntervalCount: p.Recurring.IntervalCount,
			}
			
			if plan.Name == "" {
				plan.Name = "Unnamed Plan"
			}
			if plan.Description == "" {
				plan.Description = "No description"
			}
			
			plans = append(plans, plan)
		}
	}
	
	if err := iter.Err(); err != nil {
		h.logger.Error("Error fetching plans", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}
	
	h.logger.Info("Plans fetched successfully",
		zap.Int("total_prices", totalCount),
		zap.Int("active_plans", len(plans)),
	)
	
	if len(plans) == 0 {
		return c.JSON(http.StatusOK, echo.Map{
			"plans":   []entity.Plan{},
			"message": "No active recurring prices found. Please create recurring prices in Stripe Dashboard.",
		})
	}
	
	return c.JSON(http.StatusOK, echo.Map{
		"plans": plans,
	})
}