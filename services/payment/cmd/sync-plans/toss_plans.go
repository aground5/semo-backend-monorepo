package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"gopkg.in/yaml.v3"
)

type tossPlansFile struct {
	Plans []tossPlanEntry `yaml:"plans"`
}

type tossPlanEntry struct {
	ProviderPriceID   string                 `yaml:"provider_price_id"`
	ProviderProductID string                 `yaml:"provider_product_id"`
	PgProvider        string                 `yaml:"pg_provider"`
	Currency          string                 `yaml:"currency"`
	DisplayName       string                 `yaml:"display_name"`
	Type              string                 `yaml:"type"`
	CreditsPerCycle   int                    `yaml:"credits_per_cycle"`
	Features          map[string]interface{} `yaml:"features"`
	SortOrder         int                    `yaml:"sort_order"`
	IsActive          *bool                  `yaml:"is_active"`
}

func loadTossPlansFromYAML(path string) ([]*model.PaymentPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read toss plans file: %w", err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var file tossPlansFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("unmarshal toss plans yaml: %w", err)
	}

	plans := make([]*model.PaymentPlan, 0, len(file.Plans))
	for i, entry := range file.Plans {
		if entry.ProviderPriceID == "" {
			return nil, fmt.Errorf("plans[%d]: provider_price_id is required", i)
		}
		if entry.ProviderProductID == "" {
			return nil, fmt.Errorf("plans[%d]: provider_product_id is required", i)
		}
		if entry.DisplayName == "" {
			return nil, fmt.Errorf("plans[%d]: display_name is required", i)
		}

		planType := entry.Type
		if planType == "" {
			planType = model.PlanTypeSubscription
		}

		pgProvider := entry.PgProvider
		if pgProvider == "" {
			pgProvider = "toss"
		}

		isActive := true
		if entry.IsActive != nil {
			isActive = *entry.IsActive
		}

		features := make(model.Features, len(entry.Features))
		for k, v := range entry.Features {
			features[k] = v
		}

		currency := strings.ToUpper(strings.TrimSpace(entry.Currency))
		if currency == "" && entry.Features != nil {
			if priceMap, ok := entry.Features["price"].(map[string]interface{}); ok {
				if value, ok := priceMap["currency"].(string); ok {
					currency = strings.ToUpper(strings.TrimSpace(value))
				}
			}
		}
		if currency == "" {
			currency = "KRW"
		}

		plans = append(plans, &model.PaymentPlan{
			ProviderPriceID:   entry.ProviderPriceID,
			ProviderProductID: entry.ProviderProductID,
			PgProvider:        pgProvider,
			Currency:          currency,
			DisplayName:       entry.DisplayName,
			Type:              planType,
			CreditsPerCycle:   entry.CreditsPerCycle,
			Features:          features,
			SortOrder:         entry.SortOrder,
			IsActive:          isActive,
		})
	}

	return plans, nil
}
