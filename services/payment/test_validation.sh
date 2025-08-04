#!/bin/bash

# Test validation for the credit usage API endpoint
# This script tests various validation scenarios

API_URL="http://localhost:8080/api/v1/credits"
JWT_TOKEN="your-jwt-token-here"  # Replace with actual JWT token

echo "Testing Credit Usage API Validation"
echo "===================================="
echo ""

# Test 1: Missing required fields
echo "Test 1: Missing required fields"
echo "Request: {}"
curl -X POST $API_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{}' \
  -s | jq .
echo ""

# Test 2: Empty required fields
echo "Test 2: Empty required fields"
echo 'Request: {"amount": "", "feature_name": "", "description": ""}'
curl -X POST $API_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"amount": "", "feature_name": "", "description": ""}' \
  -s | jq .
echo ""

# Test 3: Invalid amount format
echo "Test 3: Invalid amount format"
echo 'Request: {"amount": "not-a-number", "feature_name": "test", "description": "test"}'
curl -X POST $API_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"amount": "not-a-number", "feature_name": "test", "description": "test"}' \
  -s | jq .
echo ""

# Test 4: Feature name too long (>100 chars)
echo "Test 4: Feature name too long"
LONG_FEATURE=$(printf 'a%.0s' {1..101})
echo "Request: feature_name with 101 characters"
curl -X POST $API_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d "{\"amount\": \"10.50\", \"feature_name\": \"$LONG_FEATURE\", \"description\": \"test\"}" \
  -s | jq .
echo ""

# Test 5: Description too long (>500 chars)
echo "Test 5: Description too long"
LONG_DESC=$(printf 'a%.0s' {1..501})
echo "Request: description with 501 characters"
curl -X POST $API_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d "{\"amount\": \"10.50\", \"feature_name\": \"test\", \"description\": \"$LONG_DESC\"}" \
  -s | jq .
echo ""

# Test 6: Invalid UUID for idempotency key
echo "Test 6: Invalid UUID for idempotency key"
echo 'Request: {"amount": "10.50", "feature_name": "test", "description": "test", "idempotency_key": "not-a-uuid"}'
curl -X POST $API_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"amount": "10.50", "feature_name": "test", "description": "test", "idempotency_key": "not-a-uuid"}' \
  -s | jq .
echo ""

# Test 7: Valid request
echo "Test 7: Valid request"
echo 'Request: {"amount": "10.50", "feature_name": "test_feature", "description": "Testing credit usage"}'
curl -X POST $API_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"amount": "10.50", "feature_name": "test_feature", "description": "Testing credit usage"}' \
  -s | jq .
echo ""

# Test 8: Valid request with idempotency key
echo "Test 8: Valid request with idempotency key"
echo 'Request: {"amount": "5.00", "feature_name": "test", "description": "test", "idempotency_key": "550e8400-e29b-41d4-a716-446655440000"}'
curl -X POST $API_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"amount": "5.00", "feature_name": "test", "description": "test", "idempotency_key": "550e8400-e29b-41d4-a716-446655440000"}' \
  -s | jq .
echo ""

echo "Validation tests completed!"