package main

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/price"
	"github.com/stripe/stripe-go/v79/product"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

func syncStripePlans(ctx context.Context, planSync *usecase.PlanSyncService, logger *zap.Logger) (int, error) {
	logger.Info("Starting initial sync of Stripe products and prices...")

	productParams := &stripe.ProductListParams{
		Active: stripe.Bool(true),
	}
	productParams.Limit = stripe.Int64(100)

	productIter := product.List(productParams)
	productCount := 0

	for productIter.Next() {
		prod := productIter.Product()
		productCount++

		logger.Info("Syncing product",
			zap.String("product_id", prod.ID),
			zap.String("name", prod.Name))

		priceParams := &stripe.PriceListParams{
			Product: stripe.String(prod.ID),
			Active:  stripe.Bool(true),
		}

		priceIter := price.List(priceParams)
		priceCount := 0

		for priceIter.Next() {
			p := priceIter.Price()
			priceCount++

			if err := planSync.SyncPriceWithProduct(ctx, p, prod); err != nil {
				logger.Error("Failed to sync price",
					zap.String("price_id", p.ID),
					zap.Error(err))
			} else {
				priceType := "unknown"
				if p.Type == stripe.PriceTypeRecurring {
					priceType = "subscription"
				} else if p.Type == stripe.PriceTypeOneTime {
					priceType = "one_time"
				}
				logger.Info("Price synced successfully",
					zap.String("price_id", p.ID),
					zap.String("product_name", prod.Name),
					zap.String("price_type", priceType))
			}
		}

		if err := priceIter.Err(); err != nil {
			logger.Error("Error listing prices for product",
				zap.String("product_id", prod.ID),
				zap.Error(err))
		}

		logger.Info("Product sync completed",
			zap.String("product_id", prod.ID),
			zap.Int("prices_synced", priceCount))
	}

	if err := productIter.Err(); err != nil {
		return productCount, fmt.Errorf("error listing products: %w", err)
	}

	logger.Info("Stripe sync completed",
		zap.Int("products_synced", productCount))

	return productCount, nil
}
