# Stripe Webhook Configuration

## Important: Webhook Endpoint Configuration

When setting up your Stripe webhook endpoint, you need to configure it to expand certain objects to ensure user_id metadata is included in webhook payloads.

### Required Webhook Events

Enable the following events in your Stripe Dashboard:

1. `checkout.session.completed`
2. `customer.subscription.created`
3. `customer.subscription.updated`
4. `customer.subscription.deleted`
5. `invoice.payment_succeeded`
6. `invoice.payment_failed`
7. `product.created`
8. `product.updated`
9. `product.deleted`
10. `price.created`
11. `price.updated`
12. `price.deleted`

### API Version Configuration

To ensure subscription metadata is included in invoice webhooks, you may need to:

1. Use a recent Stripe API version (2023 or later recommended)
2. Configure webhook endpoint to expand nested objects

### Testing Webhook Data

Use Stripe CLI to test webhooks locally:

```bash
# Forward webhooks to your local server
stripe listen --forward-to localhost:3002/webhook

# Trigger test events
stripe trigger invoice.payment_succeeded
```

### Metadata Strategy

The system uses multiple layers of metadata to ensure user_id is preserved:

1. **Checkout Session Metadata**: Set on the session itself
2. **Subscription Metadata**: Set via `subscription_data.metadata`
3. **Customer Mapping**: Fallback storage in local database

This multi-layer approach ensures user_id can be retrieved even when Stripe doesn't expand subscription objects in webhook payloads.