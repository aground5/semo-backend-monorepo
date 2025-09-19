package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"go.uber.org/zap"
)

type PlansHandler struct {
	logger   *zap.Logger
	planRepo repository.PlanRepository
}

func NewPlansHandler(logger *zap.Logger, planRepo repository.PlanRepository) *PlansHandler {
	return &PlansHandler{
		logger:   logger,
		planRepo: planRepo,
	}
}

// GetSubscriptionPlans returns all subscription-type payment plans
func (h *PlansHandler) GetSubscriptionPlans(c echo.Context) error {
	h.logger.Info("Fetching subscription-type payment plans from database...")

	ctx := c.Request().Context()

	// Query subscription-type payment plans from database
	dbPlans, err := h.planRepo.GetByType(ctx, "subscription")
	if err != nil {
		h.logger.Error("Error fetching subscription-type payment plans from database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to fetch subscription-type payment plans",
		})
	}

	// Convert to entity format
	var plans []entity.Plan
	for _, dbPlan := range dbPlans {
		plan := entity.Plan{
			ID:            dbPlan.ProviderPriceID,
			Name:          dbPlan.DisplayName,
			Description:   "",                            // Could be added to features
			Amount:        int64(dbPlan.CreditsPerCycle), // This should be price amount, not credits
			Currency:      "KRW",                         // Default currency
			Interval:      "month",                       // Default interval
			IntervalCount: 1,
		}

		// Extract description from features if available
		if desc, ok := dbPlan.Features["description"].(string); ok {
			plan.Description = desc
		}

		// Extract price info from features if available
		if amount, ok := dbPlan.Features["amount"].(float64); ok {
			plan.Amount = int64(amount)
		}
		if currency, ok := dbPlan.Features["currency"].(string); ok {
			plan.Currency = currency
		}
		if interval, ok := dbPlan.Features["interval"].(string); ok {
			plan.Interval = interval
		}
		if intervalCount, ok := dbPlan.Features["interval_count"].(float64); ok {
			plan.IntervalCount = int64(intervalCount)
		}

		plans = append(plans, plan)
	}

	h.logger.Info("Plans fetched successfully from database",
		zap.Int("plan_count", len(plans)),
	)

	if len(plans) == 0 {
		return c.JSON(http.StatusOK, echo.Map{
			"plans":   []entity.Plan{},
			"message": "No active plans found. Waiting for Stripe webhook sync.",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"plans": plans,
	})
}

// GetOneTimePlans returns all one-time payment plans
func (h *PlansHandler) GetOneTimePlans(c echo.Context) error {
	h.logger.Info("Fetching one-time payment plans from database...")

	ctx := c.Request().Context()

	// Query one-time payment plans from database
	dbPlans, err := h.planRepo.GetByType(ctx, "one_time")
	if err != nil {
		h.logger.Error("Error fetching one-time payment plans from database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to fetch one-time payment plans",
		})
	}

	// Convert to entity format
	var plans []entity.Plan
	for _, dbPlan := range dbPlans {
		plan := entity.Plan{
			ID:          dbPlan.ProviderPriceID,
			Name:        dbPlan.DisplayName,
			Description: "",                            // Could be added to features
			Amount:      int64(dbPlan.CreditsPerCycle), // This should be price amount, not credits
			Currency:    "KRW",                         // Default currency
			Type:        "one_time",                    // One-time payment
		}

		// Extract description from features if available
		if desc, ok := dbPlan.Features["description"].(string); ok {
			plan.Description = desc
		}

		// Extract price info from features if available
		if amount, ok := dbPlan.Features["amount"].(float64); ok {
			plan.Amount = int64(amount)
		}
		if currency, ok := dbPlan.Features["currency"].(string); ok {
			plan.Currency = currency
		}

		plans = append(plans, plan)
	}

	h.logger.Info("One-time payment plans fetched successfully from database",
		zap.Int("plan_count", len(plans)),
	)

	if len(plans) == 0 {
		return c.JSON(http.StatusOK, echo.Map{
			"plans":   []entity.Plan{},
			"message": "No active one-time payment plans found. Waiting for Stripe webhook sync.",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"plans": plans,
	})
}

// GetPlans returns all plans (backward compatibility)
func (h *PlansHandler) GetPlans(c echo.Context) error {
	h.logger.Info("Fetching all plans from database...")

	ctx := c.Request().Context()

	// Query all plans from database
	dbPlans, err := h.planRepo.GetAll(ctx)
	if err != nil {
		h.logger.Error("Error fetching plans from database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to fetch plans",
		})
	}

	// Convert to entity format
	var plans []entity.Plan
	for _, dbPlan := range dbPlans {
		plan := entity.Plan{
			ID:          dbPlan.ProviderPriceID,
			Name:        dbPlan.DisplayName,
			Description: "",                            // Could be added to features
			Amount:      int64(dbPlan.CreditsPerCycle), // This should be price amount, not credits
			Currency:    "KRW",                         // Default currency
			Type:        dbPlan.Type,                   // Include the type
		}

		// Set interval for subscriptions
		if dbPlan.Type == "subscription" {
			plan.Interval = "month" // Default interval
			plan.IntervalCount = 1
		}

		// Extract description from features if available
		if desc, ok := dbPlan.Features["description"].(string); ok {
			plan.Description = desc
		}

		// Extract price info from features if available
		if amount, ok := dbPlan.Features["amount"].(float64); ok {
			plan.Amount = int64(amount)
		}
		if currency, ok := dbPlan.Features["currency"].(string); ok {
			plan.Currency = currency
		}
		if dbPlan.Type == "subscription" {
			if interval, ok := dbPlan.Features["interval"].(string); ok {
				plan.Interval = interval
			}
			if intervalCount, ok := dbPlan.Features["interval_count"].(float64); ok {
				plan.IntervalCount = int64(intervalCount)
			}
		}

		plans = append(plans, plan)
	}

	h.logger.Info("Plans fetched successfully from database",
		zap.Int("plan_count", len(plans)),
	)

	if len(plans) == 0 {
		return c.JSON(http.StatusOK, echo.Map{
			"plans":   []entity.Plan{},
			"message": "No active plans found. Waiting for Stripe webhook sync.",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"plans": plans,
	})
}
