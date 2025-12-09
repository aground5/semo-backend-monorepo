package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
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

	provider := c.QueryParam("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "provider query parameter is required",
		})
	}

	currency := strings.ToUpper(strings.TrimSpace(c.QueryParam("currency")))

	dbPlans, err := h.planRepo.GetByTypeAndProvider(ctx, model.PlanTypeSubscription, provider, currency)
	if err != nil {
		h.logger.Error("Error fetching subscription-type payment plans from database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to fetch subscription-type payment plans",
		})
	}

	plans := make([]entity.Plan, 0, len(dbPlans))
	for _, dbPlan := range dbPlans {
		plans = append(plans, mapPaymentPlanToEntity(dbPlan))
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

	provider := c.QueryParam("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "provider query parameter is required",
		})
	}

	currency := strings.ToUpper(strings.TrimSpace(c.QueryParam("currency")))

	dbPlans, err := h.planRepo.GetByTypeAndProvider(ctx, model.PlanTypeOneTime, provider, currency)
	if err != nil {
		h.logger.Error("Error fetching one-time payment plans from database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to fetch one-time payment plans",
		})
	}

	plans := make([]entity.Plan, 0, len(dbPlans))
	for _, dbPlan := range dbPlans {
		plans = append(plans, mapPaymentPlanToEntity(dbPlan))
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

	provider := c.QueryParam("provider")
	currency := strings.ToUpper(strings.TrimSpace(c.QueryParam("currency")))

	dbPlans, err := h.planRepo.GetAll(ctx)
	if provider != "" || currency != "" {
		filtered := make([]*model.PaymentPlan, 0, len(dbPlans))
		for _, plan := range dbPlans {
			if provider != "" && plan.PgProvider != provider {
				continue
			}
			if currency != "" && strings.ToUpper(plan.Currency) != currency {
				continue
			}
			filtered = append(filtered, plan)
		}
		dbPlans = filtered
	}
	if err != nil {
		h.logger.Error("Error fetching plans from database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to fetch plans",
		})
	}

	plans := make([]entity.Plan, 0, len(dbPlans))
	for _, dbPlan := range dbPlans {
		plans = append(plans, mapPaymentPlanToEntity(dbPlan))
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

func mapPaymentPlanToEntity(dbPlan *model.PaymentPlan) entity.Plan {
	plan := entity.Plan{
		ID:       dbPlan.ProviderPriceID,
		Name:     dbPlan.DisplayName,
		Type:     dbPlan.Type,
		Provider: dbPlan.PgProvider,
		Currency: dbPlan.Currency,
	}

	if plan.Type == "" {
		plan.Type = model.PlanTypeSubscription
	}

	features := dbPlan.Features
	if features == nil {
		return plan
	}
	featureMap := map[string]interface{}(features)

	if desc, ok := getString(featureMap, "description"); ok {
		plan.Description = desc
	}

	if priceMap, ok := getMap(featureMap, "price"); ok {
		var price entity.PlanPrice
		if amount, ok := getInt64(priceMap, "amount"); ok {
			plan.Amount = amount
			price.Amount = amount
		}
		if currency, ok := getString(priceMap, "currency"); ok {
			plan.Currency = currency
			price.Currency = currency
		}
		if interval, ok := getString(priceMap, "interval"); ok {
			plan.Interval = interval
		}
		if intervalCount, ok := getInt64(priceMap, "interval_count"); ok {
			plan.IntervalCount = intervalCount
		}
		if price.Amount != 0 || price.Currency != "" {
			plan.Price = &price
		}
	}

	if summaryMap, ok := getMap(featureMap, "summary"); ok {
		if summary := buildPlanSummary(summaryMap); summary != nil {
			plan.Summary = summary
		}
	}

	if badgeList, ok := getSlice(featureMap, "badges"); ok {
		plan.Badges = buildPlanBadges(badgeList)
	}

	if benefits, ok := getSlice(featureMap, "benefits"); ok {
		plan.Benefits = buildStringSlice(benefits)
	}

	if ctaMap, ok := getMap(featureMap, "cta"); ok {
		if cta := buildPlanCTA(ctaMap); cta != nil {
			plan.CTA = cta
		}
	}

	if plan.Type == model.PlanTypeSubscription {
		if plan.Interval == "" {
			plan.Interval = "month"
		}
		if plan.IntervalCount == 0 {
			plan.IntervalCount = 1
		}
	}

	return plan
}

func getString(m map[string]interface{}, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	value, exists := m[key]
	if !exists {
		return "", false
	}
	return toString(value)
}

func getInt64(m map[string]interface{}, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	value, exists := m[key]
	if !exists {
		return 0, false
	}
	return toInt64(value)
}

func getMap(m map[string]interface{}, key string) (map[string]interface{}, bool) {
	if m == nil {
		return nil, false
	}
	value, exists := m[key]
	if !exists {
		return nil, false
	}
	return toMap(value)
}

func getSlice(m map[string]interface{}, key string) ([]interface{}, bool) {
	if m == nil {
		return nil, false
	}
	value, exists := m[key]
	if !exists {
		return nil, false
	}
	return toSlice(value)
}

func toString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case int:
		return strconv.Itoa(v), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case float32:
		return strconv.FormatInt(int64(v), 10), true
	case float64:
		return strconv.FormatInt(int64(v), 10), true
	default:
		return "", false
	}
}

func toInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		cleaned := strings.ReplaceAll(strings.TrimSpace(v), ",", "")
		if cleaned == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(cleaned, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func toMap(value interface{}) (map[string]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	switch v := value.(type) {
	case map[string]interface{}:
		return v, true
	default:
		return nil, false
	}
}

func toSlice(value interface{}) ([]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	switch v := value.(type) {
	case []interface{}:
		return v, true
	default:
		return nil, false
	}
}

func buildPlanSummary(summaryMap map[string]interface{}) *entity.PlanSummary {
	summary := &entity.PlanSummary{}

	if title, ok := getString(summaryMap, "title"); ok {
		summary.Title = title
	}
	if subtitle, ok := getString(summaryMap, "subtitle"); ok {
		summary.Subtitle = subtitle
	}
	if tagline, ok := getString(summaryMap, "tagline"); ok {
		summary.Tagline = tagline
	}
	if audience, ok := getString(summaryMap, "audience"); ok {
		summary.Audience = audience
	}
	if details, ok := getString(summaryMap, "details"); ok {
		summary.Details = details
	}

	if summary.Title == "" && summary.Subtitle == "" && summary.Tagline == "" && summary.Audience == "" && summary.Details == "" {
		return nil
	}

	return summary
}

func buildPlanBadges(rawItems []interface{}) []entity.PlanBadge {
	badges := make([]entity.PlanBadge, 0, len(rawItems))
	for _, item := range rawItems {
		badgeMap, ok := toMap(item)
		if !ok {
			continue
		}

		badge := entity.PlanBadge{}
		if kind, ok := getString(badgeMap, "kind"); ok {
			badge.Kind = kind
		}
		if label, ok := getString(badgeMap, "label"); ok {
			badge.Label = label
		}

		if badge.Kind == "" && badge.Label == "" {
			continue
		}

		badges = append(badges, badge)
	}
	return badges
}

func buildStringSlice(items []interface{}) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := toString(item); ok {
			if strings.TrimSpace(str) == "" {
				continue
			}
			result = append(result, str)
		}
	}
	return result
}

func buildPlanCTA(ctaMap map[string]interface{}) *entity.PlanCTA {
	cta := &entity.PlanCTA{}
	if label, ok := getString(ctaMap, "label"); ok {
		cta.Label = label
	}
	if cta.Label == "" {
		return nil
	}
	return cta
}
