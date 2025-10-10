# Payment Service Database Migrations

This directory contains SQL migration scripts for the payment service database.

## Migrations

### 001_add_subscription_cancellation_enum.sql

**Purpose**: Adds the `subscription_cancellation` value to the `transaction_type` enum.

**When to run**: If you encounter the error:
```
ERROR: invalid input value for enum transaction_type: "subscription_cancellation" (SQLSTATE 22P02)
```

**How to run**:
```bash
# Connect to your database and run:
psql -U your_user -d payment_db -f migrations/001_add_subscription_cancellation_enum.sql

# Or using docker-compose:
docker-compose exec postgres psql -U payment_user -d payment_db -f /migrations/001_add_subscription_cancellation_enum.sql
```

**What it does**:
1. Checks if `subscription_cancellation` already exists in the `transaction_type` enum
2. If not, adds it to the enum
3. Displays all enum values for verification

**Note**: The application's migration code will try to add this automatically on startup, but if that fails (e.g., due to transaction constraints), you'll need to run this script manually.