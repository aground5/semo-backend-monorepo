-- Migration: Rename subscription_plans table to payment_plans
-- Date: 2025-01-18
-- Description: Rename subscription_plans table to payment_plans to better reflect that it contains both subscription and one-time payment plans

-- Rename the table
ALTER TABLE subscription_plans RENAME TO payment_plans;

-- Add comment to the table
COMMENT ON TABLE payment_plans IS 'Payment plans including both subscription and one-time payment options';