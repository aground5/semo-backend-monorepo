package main

import (
	"context"
	"log"

	"github.com/stripe/stripe-go/v79"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
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
	repos := database.NewRepositories(db, &cfg.Service.Supabase, logger)

	// Create plan sync service
	planSync := usecase.NewPlanSyncService(repos.Plan, logger)

	ctx := context.Background()

	stripeProductsSynced, err := syncStripePlans(ctx, planSync, logger)
	if err != nil {
		logger.Fatal("Failed to sync Stripe plans", zap.Error(err))
	}

	tossPlansSynced := 0
	tossPlanFiles := []string{}
	if cfg.Service.Toss.PlansFile != "" {
		tossPlanFiles = append(tossPlanFiles, cfg.Service.Toss.PlansFile)
	}
	if cfg.Service.Toss.USDPlansFile != "" {
		tossPlanFiles = append(tossPlanFiles, cfg.Service.Toss.USDPlansFile)
	}

	for _, planPath := range tossPlanFiles {
		logger.Info("Syncing Toss plans from YAML",
			zap.String("path", planPath))

		plans, err := loadTossPlansFromYAML(planPath)
		if err != nil {
			logger.Fatal("Failed to load Toss plans from YAML", zap.Error(err))
		}

		for _, plan := range plans {
			if plan.Features == nil {
				plan.Features = make(model.Features)
			}
			if err := repos.Plan.Upsert(ctx, plan); err != nil {
				logger.Error("Failed to upsert Toss plan",
					zap.String("provider_price_id", plan.ProviderPriceID),
					zap.Error(err))
				continue
			}
			tossPlansSynced++
		}
	}

	if tossPlansSynced > 0 {
		logger.Info("Toss plans synced",
			zap.Int("plans_synced", tossPlansSynced))
	}

	logger.Info("Initial sync completed",
		zap.Int("stripe_products_synced", stripeProductsSynced),
		zap.Int("toss_plans_synced", tossPlansSynced))
}
