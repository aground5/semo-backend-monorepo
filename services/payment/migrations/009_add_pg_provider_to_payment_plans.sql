-- Migration: Add pg_provider column to payment_plans
-- Date: 2025-02-05
-- Description: Track which payment gateway provides each plan.

ALTER TABLE payment_plans
ADD COLUMN IF NOT EXISTS pg_provider VARCHAR(50);

COMMENT ON COLUMN payment_plans.pg_provider IS 'Payment gateway providing this plan (e.g., stripe, toss).';
