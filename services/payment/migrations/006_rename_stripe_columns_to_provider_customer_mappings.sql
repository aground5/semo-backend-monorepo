-- Migration: Rename stripe columns to provider in customer_mappings table
-- Date: 2025-01-08
-- Purpose: Change column names from stripe_* to provider_* in customer_mappings table for provider agnostic naming

-- Rename columns in customer_mappings table
ALTER TABLE customer_mappings
RENAME COLUMN stripe_customer_id TO provider_customer_id;

-- Update indexes (if any exist on these columns)
-- Note: GORM unique constraints will be handled automatically

-- Add comments to document the schema change
COMMENT ON COLUMN customer_mappings.provider_customer_id IS 'Payment provider customer ID (e.g., Stripe Customer ID)';