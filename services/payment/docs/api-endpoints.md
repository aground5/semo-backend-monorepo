# Payment Service API Documentation

## Credit Management Endpoints

### Use Credits
Deduct credits from a user's balance for feature usage.

**Endpoint:** `POST /api/v1/credits`

**Authentication:** Required (JWT via Supabase)

**Request Body:**
```json
{
  "amount": "10.50",
  "feature_name": "image_generation",
  "description": "Generated 5 high-resolution images",
  "usage_metadata": {
    "images_count": 5,
    "resolution": "1920x1080",
    "model": "v2"
  },
  "idempotency_key": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Request Fields:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| amount | string | Yes | Amount of credits to deduct (decimal format) |
| feature_name | string | Yes | Name of the feature using credits (max 100 chars) |
| description | string | Yes | Description of the usage (max 500 chars) |
| usage_metadata | object | No | Additional metadata about the usage |
| idempotency_key | string (UUID) | No | Optional UUID to prevent duplicate transactions |

**Success Response (200 OK):**
```json
{
  "success": true,
  "transaction_id": 12345,
  "balance_after": "89.50",
  "message": "Credits successfully deducted"
}
```

**Error Responses:**

**400 Bad Request** - Invalid request format
```json
{
  "error": "invalid amount format"
}
```

**401 Unauthorized** - Missing or invalid JWT token
```json
{
  "error": "unauthorized"
}
```

**402 Payment Required** - Insufficient credit balance
```json
{
  "error": "insufficient_credits",
  "message": "Insufficient credit balance",
  "requested_amount": "100.00",
  "available_balance": "50.00"
}
```

**409 Conflict** - Duplicate idempotency key
```json
{
  "error": "duplicate_request",
  "message": "A request with this idempotency key has already been processed"
}
```

**500 Internal Server Error** - Server error
```json
{
  "error": "failed to process credit usage"
}
```

### Get Credit Balance
Retrieve the current credit balance for the authenticated user.

**Endpoint:** `GET /api/v1/credits`

**Authentication:** Required (JWT via Supabase)

**Success Response (200 OK):**
```json
{
  "current_balance": "150.50"
}
```

### Get Transaction History
Retrieve credit transaction history for the authenticated user.

**Endpoint:** `GET /api/v1/credits/transactions`

**Authentication:** Required (JWT via Supabase)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| limit | integer | Number of transactions to return (default: 20, max: 100) |
| offset | integer | Number of transactions to skip (default: 0) |
| start_date | string (ISO 8601) | Filter transactions after this date |
| end_date | string (ISO 8601) | Filter transactions before this date |
| transaction_type | string | Filter by type: credit_allocation, credit_usage, refund, adjustment |

**Success Response (200 OK):**
```json
{
  "transactions": [
    {
      "transaction_type": "credit_usage",
      "amount": "-10.50",
      "balance_after": "89.50",
      "description": "Generated 5 high-resolution images",
      "created_at": "2024-01-15T10:30:00Z"
    },
    {
      "transaction_type": "credit_allocation",
      "amount": "100.00",
      "balance_after": "100.00",
      "description": "Credit allocation for Pro subscription",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "total": 25,
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}
```

## Usage Examples

### Using Credits with cURL
```bash
curl -X POST https://api.example.com/api/v1/credits \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": "5.00",
    "feature_name": "text_generation",
    "description": "Generated article with 1000 words"
  }'
```

### Using Credits with Idempotency Key
```bash
curl -X POST https://api.example.com/api/v1/credits \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": "10.00",
    "feature_name": "video_processing",
    "description": "Processed 5-minute video",
    "idempotency_key": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

## Implementation Notes

1. **Atomic Operations**: All credit operations are atomic to prevent race conditions
2. **Idempotency**: Optional idempotency key support prevents duplicate transactions
3. **Audit Trail**: All transactions are logged with complete metadata
4. **Security**: JWT authentication required for all credit operations
5. **Validation**: Strict input validation including positive amount checks
6. **Error Handling**: Comprehensive error responses with appropriate HTTP status codes