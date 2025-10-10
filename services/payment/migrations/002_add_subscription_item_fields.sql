-- Migration: Add subscription item fields directly to subscriptions table
-- Date: 2025-08-18
-- Purpose: Merge SubscriptionItem fields into Subscription for single subscription per customer

-- Add new columns to subscriptions table
ALTER TABLE subscriptions
ADD COLUMN IF NOT EXISTS product_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS amount BIGINT DEFAULT 0,
ADD COLUMN IF NOT EXISTS currency VARCHAR(3) DEFAULT 'KRW',
ADD COLUMN IF NOT EXISTS interval VARCHAR(20),
ADD COLUMN IF NOT EXISTS interval_count BIGINT DEFAULT 1;

-- Update existing subscriptions with data from related plans
UPDATE subscriptions s
SET 
    product_name = sp.name,
    amount = sp.amount,
    currency = sp.currency,
    interval = sp.billing_period,
    interval_count = sp.billing_period_count
FROM subscription_plans sp
WHERE s.plan_id = sp.id
  AND s.product_name IS NULL;

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_subscriptions_product_name ON subscriptions(product_name);
CREATE INDEX IF NOT EXISTS idx_subscriptions_currency ON subscriptions(currency);

-- Add comment to document the schema change
COMMENT ON COLUMN subscriptions.product_name IS 'Name of the subscription product';
COMMENT ON COLUMN subscriptions.amount IS 'Subscription amount in smallest currency unit';
COMMENT ON COLUMN subscriptions.currency IS 'ISO 4217 currency code';
COMMENT ON COLUMN subscriptions.interval IS 'Billing interval (day, week, month, year)';
COMMENT ON COLUMN subscriptions.interval_count IS 'Number of intervals between billings';