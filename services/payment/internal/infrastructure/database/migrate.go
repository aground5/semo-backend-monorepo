package database

import (
	"fmt"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Migrate runs database migrations
func Migrate(db *gorm.DB, logger *zap.Logger) error {
	logger.Info("Running database migrations...")

	// Create extensions first
	logger.Info("Creating PostgreSQL extensions...")
	if err := createExtensions(db); err != nil {
		logger.Error("Failed to create extensions", zap.Error(err))
		return err
	}
	logger.Info("PostgreSQL extensions created successfully")

	// Create custom types BEFORE auto-migrate
	logger.Info("Creating custom PostgreSQL types...")
	if err := createCustomTypes(db); err != nil {
		logger.Error("Failed to create custom types", zap.Error(err))
		return err
	}
	logger.Info("Custom PostgreSQL types created successfully")

	// Auto-migrate all models
	logger.Info("Running GORM auto-migrations...")
	err := db.AutoMigrate(
		&model.PaymentPlan{},
		&model.Subscription{},
		&model.CreditTransaction{},
		&model.UserCreditBalance{},
		&model.Payment{},
		&model.StripeWebhookEvent{},
		&model.TossWebhookEvent{},
		&model.AuditLog{},
		&model.CustomerMapping{},
	)
	if err != nil {
		logger.Error("Failed to run migrations", zap.Error(err))
		return err
	}
	logger.Info("GORM auto-migrations completed successfully")

	// Create custom indexes and constraints
	logger.Info("Creating custom indexes...")
	if err := createCustomIndexes(db); err != nil {
		logger.Error("Failed to create custom indexes", zap.Error(err))
		return err
	}
	logger.Info("Custom indexes created successfully")

	// Create database functions and triggers from READDB.md
	logger.Info("Creating database functions...")
	if err := createDatabaseFunctions(db, logger); err != nil {
		logger.Error("Failed to create database functions", zap.Error(err))
		return err
	}
	logger.Info("Database functions created successfully")

	logger.Info("Database migrations completed successfully")
	return nil
}

// createCustomIndexes creates custom indexes that GORM doesn't handle automatically
func createCustomIndexes(db *gorm.DB) error {
	// Create unique index for active subscriptions per universal ID
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS unique_active_subscription_per_universal_id ON subscriptions (universal_id) WHERE status = 'active'`).Error; err != nil {
		return err
	}

	// Create index for webhook events
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed ON stripe_webhook_events (created_at) WHERE status IN ('pending', 'failed')`).Error; err != nil {
		return err
	}

	// Create index for credit transactions
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_credit_transactions_reference ON credit_transactions (reference_id) WHERE reference_id IS NOT NULL`).Error; err != nil {
		return err
	}

	return nil
}

// createExtensions creates required PostgreSQL extensions
func createExtensions(db *gorm.DB) error {
	// Create extensions
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		return err
	}
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`).Error; err != nil {
		return err
	}
	return nil
}

// createDatabaseFunctions creates the database functions from READDB.md
func createDatabaseFunctions(db *gorm.DB, logger *zap.Logger) error {

	// Create audit trigger function
	logger.Info("Creating audit trigger function...")
	auditFunctionSQL := `
CREATE OR REPLACE FUNCTION audit_table_changes() RETURNS TRIGGER AS $$
DECLARE
    current_universal_id UUID;
    v_universal_id UUID;
    v_record_id BIGINT;
BEGIN
    -- Try to get universal_id context from session
    BEGIN
        current_universal_id := (current_setting('app.current_universal_id', true))::UUID;
    EXCEPTION WHEN OTHERS THEN
        current_universal_id := NULL;
    END;
    
    -- If no session universal_id, try to extract from the record
    IF current_universal_id IS NULL THEN
        -- Handle different cases based on operation
        IF TG_OP = 'DELETE' THEN
            -- Try to get universal_id from OLD record
            BEGIN
                EXECUTE format('SELECT ($1).universal_id') INTO v_universal_id USING OLD;
            EXCEPTION WHEN OTHERS THEN
                v_universal_id := NULL;
            END;
            v_record_id := OLD.id;
        ELSE
            -- Try to get universal_id from NEW record
            BEGIN
                EXECUTE format('SELECT ($1).universal_id') INTO v_universal_id USING NEW;
            EXCEPTION WHEN OTHERS THEN
                v_universal_id := NULL;
            END;
            v_record_id := NEW.id;
        END IF;
        
        current_universal_id := v_universal_id;
    END IF;

    -- Perform the audit log insert
    IF TG_OP = 'DELETE' THEN
        INSERT INTO audit_log (universal_id, action, table_name, record_id, old_values, ip_address)
        VALUES (current_universal_id, 'DELETE', TG_TABLE_NAME, v_record_id, to_jsonb(OLD), inet_client_addr());
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_log (universal_id, action, table_name, record_id, old_values, new_values, ip_address)
        VALUES (current_universal_id, 'UPDATE', TG_TABLE_NAME, v_record_id, to_jsonb(OLD), to_jsonb(NEW), inet_client_addr());
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log (universal_id, action, table_name, record_id, new_values, ip_address)
        VALUES (current_universal_id, 'INSERT', TG_TABLE_NAME, v_record_id, to_jsonb(NEW), inet_client_addr());
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;`

	if err := db.Exec(auditFunctionSQL).Error; err != nil {
		logger.Error("Failed to create audit trigger function", zap.Error(err))
		return err
	}

	// Create triggers for each table
	tables := []string{"subscriptions", "credit_transactions", "payments", "customer_mappings"}
	for _, table := range tables {
		triggerSQL := fmt.Sprintf(`
CREATE TRIGGER audit_%s
    AFTER INSERT OR UPDATE OR DELETE ON %s
    FOR EACH ROW EXECUTE FUNCTION audit_table_changes();`, table, table)

		// Drop existing trigger if it exists
		dropSQL := fmt.Sprintf(`DROP TRIGGER IF EXISTS audit_%s ON %s;`, table, table)
		if err := db.Exec(dropSQL).Error; err != nil {
			logger.Warn("Failed to drop existing trigger", zap.String("table", table), zap.Error(err))
		}

		// Create new trigger
		if err := db.Exec(triggerSQL).Error; err != nil {
			logger.Error("Failed to create audit trigger", zap.String("table", table), zap.Error(err))
			return err
		}
		logger.Info("Created audit trigger", zap.String("table", table))
	}

	// Create set_universal_id_context function for RLS
	setUniversalIdContextSQL := `
CREATE OR REPLACE FUNCTION set_universal_id_context(universal_id UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_universal_id', universal_id::TEXT, true);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;`

	if err := db.Exec(setUniversalIdContextSQL).Error; err != nil {
		logger.Error("Failed to create set_universal_id_context function", zap.Error(err))
		return err
	}

	return nil
}

// createCustomTypes creates custom PostgreSQL types
func createCustomTypes(db *gorm.DB) error {
	// Check if subscription_status type exists
	var exists bool
	db.Raw(`SELECT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'subscription_status')`).Scan(&exists)
	if !exists {
		if err := db.Exec(`CREATE TYPE subscription_status AS ENUM ('active', 'inactive')`).Error; err != nil {
			return err
		}
	}

	// Check if transaction_type exists
	db.Raw(`SELECT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'transaction_type')`).Scan(&exists)
	if !exists {
		if err := db.Exec(`CREATE TYPE transaction_type AS ENUM ('credit_allocation', 'credit_usage', 'refund', 'adjustment', 'subscription_cancellation')`).Error; err != nil {
			return err
		}
	} else {
		// For existing enum, check if subscription_cancellation value exists and add it if missing
		var hasSubscriptionCancellation bool
		db.Raw(`SELECT EXISTS (
			SELECT 1 FROM pg_enum 
			WHERE enumlabel = 'subscription_cancellation' 
			AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'transaction_type')
		)`).Scan(&hasSubscriptionCancellation)
		
		if !hasSubscriptionCancellation {
			// Add the new enum value to existing type
			// Note: ALTER TYPE ADD VALUE cannot be executed inside a transaction block in some PostgreSQL versions
			if err := db.Exec(`ALTER TYPE transaction_type ADD VALUE IF NOT EXISTS 'subscription_cancellation'`).Error; err != nil {
				// This might fail if we're inside a transaction
				// Try to commit first and retry
				_ = db.Exec(`COMMIT`).Error // Ignore error, might not be in a transaction
				if err := db.Exec(`ALTER TYPE transaction_type ADD VALUE IF NOT EXISTS 'subscription_cancellation'`).Error; err != nil {
					// If it still fails, just log and continue
					// The application might be running with an older database
					// Users should run the migration script manually: migrations/001_add_subscription_cancellation_enum.sql
				}
			}
		}
	}

	// Check if webhook_status exists
	db.Raw(`SELECT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'webhook_status')`).Scan(&exists)
	if !exists {
		if err := db.Exec(`CREATE TYPE webhook_status AS ENUM ('pending', 'processing', 'completed', 'failed')`).Error; err != nil {
			return err
		}
	}

	return nil
}
