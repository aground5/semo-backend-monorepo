-- Migration: Add currency column to payment_plans

ALTER TABLE payment_plans
    ADD COLUMN IF NOT EXISTS currency VARCHAR(10) NOT NULL DEFAULT 'KRW';

UPDATE payment_plans
SET currency = UPPER(COALESCE(features -> 'price' ->> 'currency', currency, 'KRW'))
WHERE currency IS NULL OR currency = 'KRW';
