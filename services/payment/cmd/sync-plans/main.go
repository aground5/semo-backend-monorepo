package main

import (
	"context"
	"log"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/product"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/infrastructure/database"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize Stripe
	stripe.Key = cfg.Service.StripeSecretKey

	// Initialize database connection
	db, err := database.NewConnection(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := database.Close(db, logger); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		}
	}()

	// Run migrations
	if err := database.Migrate(db, logger); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	// Initialize repositories
	repos := database.NewRepositories(db, logger)

	// Create plan sync service
	planSync := usecase.NewPlanSyncService(repos.Plan, logger)

	ctx := context.Background()

	logger.Info("Starting initial sync of Stripe products and prices...")

	// Sync all active products
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

		// Get all prices for this product
		priceParams := &stripe.PriceListParams{
			Product: stripe.String(prod.ID),
			Active:  stripe.Bool(true),
		}

		priceIter := price.List(priceParams)
		priceCount := 0

		for priceIter.Next() {
			p := priceIter.Price()
			priceCount++

			// Sync all price types (subscriptions and one-time payments)
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
		logger.Fatal("Error listing products", zap.Error(err))
	}

	logger.Info("Initial sync completed",
		zap.Int("products_synced", productCount))
}
