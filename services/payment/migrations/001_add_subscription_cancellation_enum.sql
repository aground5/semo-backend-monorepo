-- Migration to add subscription_cancellation to transaction_type enum
-- Run this script to update existing databases

-- Check if the enum value already exists before adding it
DO $$ 
BEGIN
    -- Check if subscription_cancellation value exists in transaction_type enum
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_enum 
        WHERE enumlabel = 'subscription_cancellation' 
        AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'transaction_type')
    ) THEN
        -- Add the new value to the enum
        ALTER TYPE transaction_type ADD VALUE 'subscription_cancellation';
        RAISE NOTICE 'Added subscription_cancellation to transaction_type enum';
    ELSE
        RAISE NOTICE 'subscription_cancellation already exists in transaction_type enum';
    END IF;
END $$;

-- Verify the enum values
SELECT unnest(enum_range(NULL::transaction_type)) AS transaction_type_values;