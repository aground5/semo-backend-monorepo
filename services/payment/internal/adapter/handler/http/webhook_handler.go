package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/adapter/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	logger              *zap.Logger
	webhookSecret       string
	webhookRepo         repository.WebhookRepository
	subscriptionRepo    domainRepo.SubscriptionRepository
	paymentRepo         domainRepo.PaymentRepository
	customerMappingRepo domainRepo.CustomerMappingRepository
	creditService       *usecase.CreditService
	planSyncService     *usecase.PlanSyncService
	serviceProvider     string
	subscriptions       map[string]*entity.Subscription
	payments            []PaymentData
	mu                  sync.RWMutex
}

type PaymentData struct {
	InvoiceID      string
	CustomerID     string
	SubscriptionID string
	Amount         int64
	Status         string
	CreatedAt      time.Time
}

func NewWebhookHandler(logger *zap.Logger, webhookSecret string, webhookRepo repository.WebhookRepository, subscriptionRepo domainRepo.SubscriptionRepository, paymentRepo domainRepo.PaymentRepository, customerMappingRepo domainRepo.CustomerMappingRepository, creditRepo domainRepo.CreditRepository, planRepo repository.PlanRepository, serviceProvider string) *WebhookHandler {
	planSyncService := usecase.NewPlanSyncService(planRepo, logger)
	creditService := usecase.NewCreditService(creditRepo, subscriptionRepo, planRepo, logger, serviceProvider)

	return &WebhookHandler{
		logger:              logger,
		webhookSecret:       webhookSecret,
		webhookRepo:         webhookRepo,
		subscriptionRepo:    subscriptionRepo,
		paymentRepo:         paymentRepo,
		customerMappingRepo: customerMappingRepo,
		creditService:       creditService,
		planSyncService:     planSyncService,
		serviceProvider:     serviceProvider,
		subscriptions:       make(map[string]*entity.Subscription),
		payments:            make([]PaymentData, 0),
	}
}

func (h *WebhookHandler) GetWebhookData(c echo.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return c.JSON(http.StatusOK, echo.Map{
		"subscriptions": h.subscriptions,
		"payments":      h.payments,
		"payment_count": len(h.payments),
	})
}

func (h *WebhookHandler) HandleWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Error("Error reading request body", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error reading request body"})
	}

	sig := c.Request().Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEventWithOptions(
		body,
		sig,
		h.webhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)

	if err != nil {
		h.logger.Error("Webhook signature verification failed",
			zap.Error(err),
			zap.String("signature", sig),
		)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Webhook signature verification failed: " + err.Error(),
		})
	}

	h.logger.Info("Webhook Event Received",
		zap.String("type", string(event.Type)),
		zap.String("id", event.ID),
		zap.Time("created", time.Unix(event.Created, 0)),
	)

	// Save webhook event to database
	if h.webhookRepo != nil {
		if err := h.webhookRepo.SaveEvent(c.Request().Context(), event.ID, string(event.Type), event.Data.Raw); err != nil {
			h.logger.Error("Failed to save webhook event", zap.Error(err))
		}
	}

	switch event.Type {
	case stripe.EventTypeSetupIntentSucceeded:
		var setupIntent stripe.SetupIntent
		if err := json.Unmarshal(event.Data.Raw, &setupIntent); err != nil {
			h.logger.Error("Error parsing setup intent", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}

		h.logger.Info("SETUP INTENT SUCCEEDED",
			zap.String("setup_intent_id", setupIntent.ID),
			zap.String("payment_method", setupIntent.PaymentMethod.ID),
		)

		// Customer가 없으면 처리하지 않음
		if setupIntent.Customer == nil || setupIntent.Customer.ID == "" {
			h.logger.Info("No customer associated with setup intent")
			return c.JSON(http.StatusOK, echo.Map{"received": true})
		}

		// Extract user_id, email, and payment mode from metadata
		var universalID string
		var userEmail string
		var paymentMode string // "subscription" or "payment"

		if setupIntent.Metadata != nil {
			if uid, ok := setupIntent.Metadata["user_id"]; ok {
				universalID = uid
				h.logger.Info("Found user_id in setup intent metadata",
					zap.String("universal_id", universalID),
					zap.String("setup_intent_id", setupIntent.ID))
			}

			// metadata에서 email 정보 추출 (프론트엔드에서 설정 필요)
			if email, ok := setupIntent.Metadata["email"]; ok {
				userEmail = email
			}

			// metadata에서 mode 정보 추출
			if mode, ok := setupIntent.Metadata["mode"]; ok {
				paymentMode = mode
			}
		}

		customerID := setupIntent.Customer.ID

		h.logger.Info("Setup intent details",
			zap.String("customer_id", customerID),
			zap.String("email", userEmail),
			zap.String("mode", paymentMode),
		)

		if paymentMode == "payment" {
			// 일회성 결제일 때만 여기서 CustomerMapping 저장
			if universalID != "" && isValidUUID(universalID) && h.customerMappingRepo != nil {
				customerMapping := &entity.CustomerMapping{
					Provider:           stripeProvider,
					ProviderCustomerID: customerID,
					UniversalID:        universalID,
					Email:              userEmail,
				}

				h.logger.Info("Creating customer mapping from one-time payment",
					zap.String("customer_id", customerID),
					zap.String("universal_id", universalID),
					zap.String("email", userEmail))

				// Check if mapping already exists
				existing, _ := h.customerMappingRepo.GetByProviderCustomerID(c.Request().Context(), stripeProvider, customerID)
				if existing == nil {
					if err := h.customerMappingRepo.Create(c.Request().Context(), customerMapping); err != nil {
						h.logger.Error("Failed to save customer mapping",
							zap.String("customer_id", customerID),
							zap.String("universal_id", universalID),
							zap.String("email", userEmail),
							zap.Error(err))
					} else {
						h.logger.Info("Customer mapping saved successfully",
							zap.String("customer_id", customerID),
							zap.String("universal_id", universalID),
							zap.String("email", userEmail))
					}
				} else {
					h.logger.Info("Customer mapping already exists",
						zap.String("customer_id", customerID),
						zap.String("existing_email", existing.Email))
				}
			}
		} else {
			h.logger.Warn("Unknown payment mode or mode not specified",
				zap.String("mode", paymentMode),
				zap.String("customer_id", customerID))
		}

		return c.JSON(http.StatusOK, echo.Map{"received": true})

	case stripe.EventTypeCustomerSubscriptionCreated, stripe.EventTypeCustomerSubscriptionUpdated:
		var rawData map[string]interface{}
		if err := json.Unmarshal(event.Data.Raw, &rawData); err != nil {
			h.logger.Error("Error parsing raw subscription data", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}

		h.logger.Info("Complete webhook data structure",
			zap.String("event_type", string(event.Type)),
			zap.Any("raw_data", rawData),
			zap.Any("items", rawData["items"]),
			zap.Any("metadata", rawData["metadata"]),
			zap.Any("customer", rawData["customer"]),
			zap.Any("status", rawData["status"]),
			zap.Any("current_period_end", rawData["current_period_end"]))

		subscriptionID, _ := rawData["id"].(string)
		status, _ := rawData["status"].(string)
		customerID, _ := rawData["customer"].(string)

		currentPeriodEnd := int64(0)
		if cpe, ok := rawData["current_period_end"].(float64); ok {
			currentPeriodEnd = int64(cpe)
		}

		var productID string
		if items, ok := rawData["items"].(map[string]interface{}); ok {
			if data, ok := items["data"].([]interface{}); ok && len(data) > 0 {
				if item, ok := data[0].(map[string]interface{}); ok {
					if price, ok := item["price"].(map[string]interface{}); ok {
						if product, ok := price["product"].(string); ok {
							productID = product
							h.logger.Info("Extracted product ID for plan_id",
								zap.String("product_id", productID))
						}
					}
				}
			}
		}

		h.logger.Info("SUBSCRIPTION CREATED/UPDATED",
			zap.String("subscription_id", subscriptionID),
			zap.String("customer_id", customerID),
			zap.String("status", status),
			zap.Time("period_end", time.Unix(currentPeriodEnd, 0)),
		)

		if customerID != "" {
			// Extract user ID from metadata
			var universalID string
			var customerEmail string

			if metadata, ok := rawData["metadata"].(map[string]interface{}); ok {
				universalID, _ = metadata["user_id"].(string)
				h.logger.Info("Extracted user ID from subscription metadata",
					zap.String("universal_id", universalID),
					zap.String("subscription_id", subscriptionID))
			} else {
				h.logger.Warn("No metadata found in subscription data",
					zap.String("subscription_id", subscriptionID))
			}

			// Try to extract customer email if customer object is expanded
			if customer, ok := rawData["customer"].(map[string]interface{}); ok {
				h.logger.Debug("Customer object is expanded in webhook",
					zap.Any("customer_data", customer))

				// Check for email in expanded customer object
				if email, ok := customer["email"].(string); ok && email != "" {
					customerEmail = email
					h.logger.Info("Extracted customer email from expanded customer object",
						zap.String("email", customerEmail),
						zap.String("customer_id", customerID))
				} else {
					h.logger.Warn("No email found in expanded customer object",
						zap.Any("customer_keys", getMapKeys(customer)))
				}

				// Also check customer metadata for user_id if not found in subscription
				if universalID == "" {
					if customerMeta, ok := customer["metadata"].(map[string]interface{}); ok {
						universalID, _ = customerMeta["user_id"].(string)
						h.logger.Info("Extracted user ID from customer metadata",
							zap.String("universal_id", universalID),
							zap.String("customer_id", customerID))
					}
				}
			} else {
				h.logger.Warn("Customer object is not expanded in webhook, only have customer ID",
					zap.String("customer_id", customerID),
					zap.String("raw_customer_type", fmt.Sprintf("%T", rawData["customer"])))
			}

			subscription := &entity.Subscription{
				ID:               subscriptionID,
				CustomerID:       customerID,
				CustomerEmail:    customerEmail,
				Status:           status,
				CurrentPeriodEnd: time.Unix(currentPeriodEnd, 0),
				PlanID:           &productID,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}

			// If we have a valid user ID, save customer mapping
			if universalID != "" && isValidUUID(universalID) && h.customerMappingRepo != nil {
				h.logger.Info("Valid user ID found for subscription",
					zap.String("universal_id", universalID),
					zap.String("customer_id", customerID),
					zap.String("subscription_id", subscriptionID))

				// Check if mapping already exists
				existing, _ := h.customerMappingRepo.GetByProviderCustomerID(c.Request().Context(), stripeProvider, customerID)
				if existing == nil {
					customerMapping := &entity.CustomerMapping{
						Provider:           stripeProvider,
						ProviderCustomerID: customerID,
						UniversalID:        universalID,
						Email:              customerEmail, // Use the extracted email
					}

					if err := h.customerMappingRepo.Create(c.Request().Context(), customerMapping); err != nil {
						h.logger.Error("Failed to save customer mapping",
							zap.String("customer_id", customerID),
							zap.String("universal_id", universalID),
							zap.Error(err))
					} else {
						h.logger.Info("Customer mapping saved from subscription webhook",
							zap.String("customer_id", customerID),
							zap.String("universal_id", universalID),
							zap.String("email", customerEmail))
					}
				} else if existing != nil && existing.Email == "" && customerEmail != "" {
					existing.Email = customerEmail
					if err := h.customerMappingRepo.Update(c.Request().Context(), existing); err != nil {
						h.logger.Error("Failed to update customer mapping with email",
							zap.String("customer_id", customerID),
							zap.String("email", customerEmail),
							zap.Error(err))
					} else {
						h.logger.Info("Updated customer mapping with email",
							zap.String("customer_id", customerID),
							zap.String("email", customerEmail))
					}
				}
			}

			// Extract subscription item data and map directly to subscription fields
			if items, ok := rawData["items"].(map[string]interface{}); ok {
				if data, ok := items["data"].([]interface{}); ok && len(data) > 0 {
					if item, ok := data[0].(map[string]interface{}); ok {
						if price, ok := item["price"].(map[string]interface{}); ok {
							// Extract product name
							if product, ok := price["product"].(string); ok {
								subscription.ProductName = product
							}

							// Extract amount
							if unitAmount, ok := price["unit_amount"].(float64); ok {
								subscription.Amount = int64(unitAmount)
							}

							// Extract currency
							if currency, ok := price["currency"].(string); ok {
								subscription.Currency = currency
							} else {
								subscription.Currency = "KRW" // Default
							}

							// Extract recurring interval information
							if recurring, ok := price["recurring"].(map[string]interface{}); ok {
								if interval, ok := recurring["interval"].(string); ok {
									subscription.Interval = interval
								}
								if intervalCount, ok := recurring["interval_count"].(float64); ok {
									subscription.IntervalCount = int64(intervalCount)
								}
							}
						}
					}
				}
			}

			// Save to database
			if h.subscriptionRepo != nil {
				ctx := c.Request().Context()
				var err error

				// 더 엄격한 중복 체크
				existing, err := h.subscriptionRepo.GetByID(ctx, subscriptionID)
				if err != nil {
					h.logger.Error("Failed to check existing subscription", zap.Error(err))
					return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
				}

				if existing != nil {
					h.logger.Info("Subscription already exists, updating...",
						zap.String("subscription_id", subscriptionID))
					err = h.subscriptionRepo.Update(ctx, subscription)
				} else {
					h.logger.Info("Creating new subscription",
						zap.String("subscription_id", subscriptionID))
					err = h.subscriptionRepo.Save(ctx, subscription)
				}

				if err != nil {
					h.logger.Error("Failed to save subscription to database",
						zap.String("subscription_id", subscriptionID),
						zap.Error(err))
				} else {
					h.logger.Info("Subscription saved to database",
						zap.String("customer_id", customerID),
						zap.String("subscription_id", subscriptionID),
						zap.Time("period_end", time.Unix(currentPeriodEnd, 0)),
					)
				}
			}

			// Also update in-memory map for backward compatibility
			h.mu.Lock()
			h.subscriptions[customerID] = subscription
			h.mu.Unlock()
		}

	case stripe.EventTypeCustomerSubscriptionDeleted:
		var rawData map[string]interface{}
		if err := json.Unmarshal(event.Data.Raw, &rawData); err != nil {
			h.logger.Error("Error parsing subscription deletion", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}

		// Extract both customer ID and subscription ID
		customerID, _ := rawData["customer"].(string)
		subscriptionID, _ := rawData["id"].(string)

		h.logger.Info("SUBSCRIPTION DELETED",
			zap.String("customer_id", customerID),
			zap.String("subscription_id", subscriptionID),
		)

		// Call Cancel function to properly handle database updates
		if subscriptionID != "" && h.subscriptionRepo != nil {
			ctx := c.Request().Context()
			if err := h.subscriptionRepo.Cancel(ctx, subscriptionID); err != nil {
				h.logger.Error("Failed to cancel subscription in database",
					zap.String("subscription_id", subscriptionID),
					zap.String("customer_id", customerID),
					zap.Error(err))
				// Note: We don't return error to Stripe to prevent webhook retries
			} else {
				h.logger.Info("Subscription successfully canceled in database",
					zap.String("subscription_id", subscriptionID),
					zap.String("customer_id", customerID))
			}
		}

		// Update in-memory state for backward compatibility
		if customerID != "" {
			h.mu.Lock()
			if sub, exists := h.subscriptions[customerID]; exists {
				sub.Status = "canceled"
				sub.UpdatedAt = time.Now()
			}
			h.mu.Unlock()
		}

	case stripe.EventTypeInvoicePaid:
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			h.logger.Error("Error parsing invoice", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}

		// Extract subscription ID from raw data if not in invoice object
		var extractedSubscriptionID string
		if invoice.Subscription != nil {
			extractedSubscriptionID = invoice.Subscription.ID
		} else {
			// Try to extract from raw data
			var rawInvoice map[string]interface{}
			if err := json.Unmarshal(event.Data.Raw, &rawInvoice); err == nil {
				// Try multiple paths to find subscription ID
				// Path 1: Direct subscription field
				if subValue, ok := rawInvoice["subscription"]; ok {
					if subID, ok := subValue.(string); ok && subID != "" {
						extractedSubscriptionID = subID
						h.logger.Info("Extracted subscription ID from raw invoice data (direct field)",
							zap.String("subscription_id", extractedSubscriptionID))
					} else if subObj, ok := subValue.(map[string]interface{}); ok {
						if id, ok := subObj["id"].(string); ok {
							extractedSubscriptionID = id
							h.logger.Info("Extracted subscription ID from subscription object",
								zap.String("subscription_id", extractedSubscriptionID))
						}
					}
				}

				// Path 2: parent.subscription_item_details.subscription
				if extractedSubscriptionID == "" {
					if parent, ok := rawInvoice["parent"].(map[string]interface{}); ok {
						if subItemDetails, ok := parent["subscription_item_details"].(map[string]interface{}); ok {
							if subValue, ok := subItemDetails["subscription"]; ok {
								if subID, ok := subValue.(string); ok && subID != "" {
									extractedSubscriptionID = subID
									h.logger.Info("Extracted subscription ID from parent.subscription_item_details.subscription",
										zap.String("subscription_id", extractedSubscriptionID))
								}
							}
						}
					}
				}

				// Path 3: parent.subscription_details.subscription
				if extractedSubscriptionID == "" {
					if parent, ok := rawInvoice["parent"].(map[string]interface{}); ok {
						if subDetails, ok := parent["subscription_details"].(map[string]interface{}); ok {
							if subValue, ok := subDetails["subscription"]; ok {
								if subID, ok := subValue.(string); ok && subID != "" {
									extractedSubscriptionID = subID
									h.logger.Info("Extracted subscription ID from parent.subscription_details.subscription",
										zap.String("subscription_id", extractedSubscriptionID))
								}
							}
						}
					}
				}

				// Path 4: Check in line items for subscription field
				if extractedSubscriptionID == "" {
					if lines, ok := rawInvoice["lines"].(map[string]interface{}); ok {
						if data, ok := lines["data"].([]interface{}); ok && len(data) > 0 {
							if lineItem, ok := data[0].(map[string]interface{}); ok {
								if subValue, ok := lineItem["subscription"]; ok {
									if subID, ok := subValue.(string); ok && subID != "" {
										extractedSubscriptionID = subID
										h.logger.Info("Extracted subscription ID from line item",
											zap.String("subscription_id", extractedSubscriptionID))
									}
								}
								// Also check subscription_item field
								if extractedSubscriptionID == "" {
									if subItem, ok := lineItem["subscription_item"]; ok {
										if subItemStr, ok := subItem.(string); ok && subItemStr != "" {
											// subscription_item might contain the subscription ID
											h.logger.Debug("Found subscription_item in line item",
												zap.String("subscription_item", subItemStr))
										}
									}
								}
							}
						}
					}
				}

				// Log what paths we've checked if still no subscription ID
				if extractedSubscriptionID == "" {
					h.logger.Warn("Could not find subscription ID in any known paths",
						zap.Bool("has_subscription_field", rawInvoice["subscription"] != nil),
						zap.Bool("has_parent", rawInvoice["parent"] != nil))

					// Debug: Log the structure to find where subscription ID might be
					if parent, ok := rawInvoice["parent"]; ok && parent != nil {
						if parentMap, ok := parent.(map[string]interface{}); ok {
							h.logger.Debug("Parent object structure", zap.Any("parent_keys", getMapKeys(parentMap)))
						}
					}

					// Also check and log top-level keys
					h.logger.Debug("Top-level invoice keys", zap.Any("keys", getMapKeys(rawInvoice)))
				}
			}
		}

		// Log raw data for debugging
		h.logger.Info("=== CREDIT ALLOCATION FLOW ANALYSIS START ===",
			zap.String("invoice_id", invoice.ID),
			zap.Int64("amount_paid", invoice.AmountPaid),
			zap.String("currency", string(invoice.Currency)))
		h.logger.Debug("Raw invoice webhook data", zap.ByteString("raw_data", event.Data.Raw))

		h.logger.Info("PAYMENT SUCCEEDED",
			zap.String("invoice_id", invoice.ID),
			zap.Int64("amount_paid", invoice.AmountPaid),
			zap.Bool("has_subscription", invoice.Subscription != nil),
			zap.Bool("has_customer", invoice.Customer != nil),
			zap.String("extracted_subscription_id", extractedSubscriptionID),
		)

		// Extract user ID from various sources
		var universalID string
		customerID := ""

		// Get customer ID
		if invoice.Customer != nil {
			customerID = invoice.Customer.ID
		}

		// Try to get user ID from invoice metadata first
		if invoice.Metadata != nil {
			if uid, ok := invoice.Metadata["user_id"]; ok {
				universalID = uid
				h.logger.Info("Found user ID in invoice metadata",
					zap.String("universal_id", uid),
					zap.String("invoice_id", invoice.ID))
			}
		}

		// If not in invoice metadata, try subscription metadata
		if universalID == "" && invoice.Subscription != nil {
			h.logger.Info("Attempting to extract user_id from subscription metadata",
				zap.String("subscription_id", invoice.Subscription.ID))

			// Try to get user ID from subscription metadata in raw data
			var rawInvoice map[string]interface{}
			if err := json.Unmarshal(event.Data.Raw, &rawInvoice); err == nil {
				// Check if subscription is a string (ID only) or object
				subValue, hasSubscription := rawInvoice["subscription"]
				h.logger.Debug("Subscription value type",
					zap.String("type", fmt.Sprintf("%T", subValue)),
					zap.Bool("has_subscription", hasSubscription))

				// If subscription is just an ID string, we need to fetch from our DB
				if subID, ok := subValue.(string); ok && subID != "" {
					h.logger.Info("Subscription is ID only, fetching from database",
						zap.String("subscription_id", subID))

					// Try to get user_id from our subscription record
					if h.subscriptionRepo != nil {
						if sub, err := h.subscriptionRepo.GetByID(c.Request().Context(), subID); err == nil && sub != nil {
							// The subscription entity should have user_id stored
							h.logger.Info("Found subscription in database",
								zap.String("subscription_id", subID))
						}
					}
				} else if subData, ok := subValue.(map[string]interface{}); ok {
					// Subscription is expanded object
					if subMeta, ok := subData["metadata"].(map[string]interface{}); ok {
						if uid, ok := subMeta["user_id"].(string); ok && isValidUUID(uid) {
							universalID = uid
							h.logger.Info("Found user ID in subscription metadata",
								zap.String("universal_id", uid),
								zap.String("subscription_id", invoice.Subscription.ID))
						} else {
							h.logger.Warn("No valid user_id in subscription metadata",
								zap.Any("metadata", subMeta))
						}
					} else {
						h.logger.Warn("No metadata in subscription object",
							zap.Any("subscription", subData))
					}
				}
			} else {
				h.logger.Error("Failed to unmarshal raw invoice data", zap.Error(err))
			}
		}

		// If still no user ID, try customer mapping
		if (universalID == "" || !isValidUUID(universalID)) && customerID != "" && h.customerMappingRepo != nil {
			h.logger.Info("Attempting to find user ID from customer mapping",
				zap.String("customer_id", customerID))

			mapping, err := h.customerMappingRepo.GetByProviderCustomerID(c.Request().Context(), stripeProvider, customerID)
			if err != nil {
				h.logger.Error("Error fetching customer mapping",
					zap.String("customer_id", customerID),
					zap.Error(err))
			} else if mapping != nil {
				universalID = mapping.UniversalID
				h.logger.Info("Found user ID from customer mapping",
					zap.String("universal_id", universalID),
					zap.String("customer_id", customerID))
			}
		}

		// Validate user ID - REQUIRED for payment processing
		if universalID == "" || !isValidUUID(universalID) {
			h.logger.Error("CRITICAL: Payment cannot be processed without valid user UUID",
				zap.String("invoice_id", invoice.ID),
				zap.String("customer_id", customerID),
				zap.String("customer_email", invoice.CustomerEmail),
				zap.String("extracted_user_id", universalID),
				zap.Bool("is_valid_uuid", isValidUUID(universalID)),
				zap.Int64("amount_paid", invoice.AmountPaid))

			// Mark webhook as failed - this will cause Stripe to retry
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error":   "Payment processing failed: User UUID is required",
				"details": "No valid user_id found in invoice, subscription metadata, or customer mapping",
			})
		}

		// Save payment to database with validated user ID
		if h.paymentRepo != nil && invoice.Customer != nil {
			paymentEntity := &entity.Payment{
				UniversalID:   universalID,
				TransactionID: invoice.ID,
				Amount:        float64(invoice.AmountPaid) / 100, // Convert cents to currency units
				Currency:      string(invoice.Currency),
				Status:        entity.PaymentStatusCompleted,
				Method:        entity.PaymentMethodCard,
				Metadata: map[string]interface{}{
					"provider_invoice_id":  invoice.ID,
					"provider_customer_id": invoice.Customer.ID,
				},
			}

			if extractedSubscriptionID != "" {
				paymentEntity.Metadata["provider_subscription_id"] = extractedSubscriptionID
			}

			if invoice.PaymentIntent != nil {
				paymentEntity.TransactionID = invoice.PaymentIntent.ID
				paymentEntity.Metadata["provider_payment_intent_id"] = invoice.PaymentIntent.ID
			}

			if err := h.paymentRepo.Create(c.Request().Context(), paymentEntity); err != nil {
				h.logger.Error("Failed to save payment to database",
					zap.String("invoice_id", invoice.ID),
					zap.String("universal_id", universalID),
					zap.Error(err))

				// Return error to make Stripe retry
				return c.JSON(http.StatusInternalServerError, echo.Map{
					"error": "Failed to save payment to database",
				})
			}

			h.logger.Info("Payment saved to database successfully",
				zap.String("payment_id", paymentEntity.ID),
				zap.String("universal_id", universalID),
				zap.Float64("amount", paymentEntity.Amount))

			// CREDIT ALLOCATION ANALYSIS - Check preconditions
			h.logger.Info("Credit allocation precondition check",
				zap.Bool("has_credit_service", h.creditService != nil),
				zap.Bool("has_invoice_lines", invoice.Lines != nil),
				zap.Int("line_items_count", func() int {
					if invoice.Lines != nil {
						return len(invoice.Lines.Data)
					}
					return 0
				}()),
				zap.String("universal_id", universalID))

			// Allocate credits for the payment
			if h.creditService != nil && invoice.Lines != nil && len(invoice.Lines.Data) > 0 {
				h.logger.Info("Starting credit allocation process",
					zap.String("invoice_id", invoice.ID),
					zap.String("universal_id", universalID))

				// Use the subscription ID we extracted earlier
				subscriptionID := extractedSubscriptionID

				// Extract product metadata from the raw invoice data
				var rawInvoice map[string]interface{}
				if err := json.Unmarshal(event.Data.Raw, &rawInvoice); err == nil {
					h.logger.Info("Successfully parsed raw invoice data for credit allocation")

					// Try to get metadata from the line items
					h.logger.Info("Analyzing line items for credit metadata")
					if lines, ok := rawInvoice["lines"].(map[string]interface{}); ok {
						h.logger.Info("Found lines object in raw invoice data")
						if data, ok := lines["data"].([]interface{}); ok && len(data) > 0 {
							h.logger.Info("Found line items data", zap.Int("count", len(data)))
							if lineItem, ok := data[0].(map[string]interface{}); ok {
								h.logger.Debug("Processing first line item", zap.Any("line_item", lineItem))
								// Get price information - navigate through pricing -> price_details -> price
								var stripePriceID string
								if pricing, ok := lineItem["pricing"].(map[string]interface{}); ok {
									h.logger.Info("Found pricing object in line item")
									if priceDetails, ok := pricing["price_details"].(map[string]interface{}); ok {
										h.logger.Info("Found price_details in pricing object")
										// Note: price is a string, not an object
										if priceID, ok := priceDetails["price"].(string); ok {
											stripePriceID = priceID
											h.logger.Info("Extracted Stripe price ID", zap.String("price_id", stripePriceID))
										} else {
											h.logger.Warn("No price string found in price_details")
										}
									} else {
										h.logger.Warn("No price_details found in pricing object")
									}
								} else {
									h.logger.Warn("No pricing object found in line item", zap.Any("line_item_keys", getMapKeys(lineItem)))
								}

								// Check if we have the product object directly in lineItem (it might be expanded there)
								if stripePriceID != "" {
									// First check if product is expanded in the lineItem itself
									if product, ok := lineItem["product"].(map[string]interface{}); ok {
										h.logger.Info("Product is expanded in line item")
										h.logger.Debug("Product data", zap.Any("product", product))
										// Product is expanded, check for metadata
										if metadata, ok := product["metadata"].(map[string]interface{}); ok {
											h.logger.Info("Found product metadata", zap.Any("metadata", metadata))

											// Log all available keys in metadata for debugging
											metadataKeys := make([]string, 0, len(metadata))
											for key := range metadata {
												metadataKeys = append(metadataKeys, key)
											}
											h.logger.Info("Available metadata keys",
												zap.Strings("keys", metadataKeys),
												zap.Int("key_count", len(metadataKeys)))

											if creditsStr, ok := metadata["credits_per_cycle"].(string); ok {
												h.logger.Info("Found credits_per_cycle in metadata", zap.String("credits_str", creditsStr))
												var credits int
												n, err := fmt.Sscanf(creditsStr, "%d", &credits)
												if err != nil || n != 1 {
													h.logger.Error("Failed to parse credits_per_cycle",
														zap.String("credits_str", creditsStr),
														zap.Error(err),
														zap.Int("parsed_count", n))
												} else {
													h.logger.Info("Successfully parsed credits from metadata", zap.Int("credits", credits))
												}

												productName := "Subscription"
												if name, ok := product["name"].(string); ok {
													productName = name
													h.logger.Info("Extracted product name", zap.String("product_name", productName))
												}

												// Allocate credits using metadata
												h.logger.Info("ATTEMPTING CREDIT ALLOCATION WITH METADATA",
													zap.String("invoice_id", invoice.ID),
													zap.String("universal_id", universalID),
													zap.Int("credits", credits),
													zap.String("product_name", productName))

												if err := h.creditService.AllocateCreditsWithMetadata(
													c.Request().Context(),
													uuid.MustParse(universalID),
													invoice.ID,
													credits,
													productName,
												); err != nil {
													h.logger.Error("CREDIT ALLOCATION FROM METADATA FAILED",
														zap.String("invoice_id", invoice.ID),
														zap.String("universal_id", universalID),
														zap.Int("credits", credits),
														zap.String("product_name", productName),
														zap.Error(err))
												} else {
													h.logger.Info("CREDIT ALLOCATION FROM METADATA SUCCESSFUL",
														zap.String("invoice_id", invoice.ID),
														zap.String("universal_id", universalID),
														zap.Int("credits", credits),
														zap.String("product_name", productName))
												}
											} else {
												h.logger.Warn("No credits_per_cycle found in product metadata",
													zap.Any("available_keys", func() []string {
														keys := make([]string, 0, len(metadata))
														for k := range metadata {
															keys = append(keys, k)
														}
														return keys
													}()))
											}
										} else {
											h.logger.Warn("No metadata found in expanded product object")
										}
									} else {
										// Product is not expanded, try to get from our database
										h.logger.Info("Product not expanded, attempting credit allocation from database",
											zap.String("price_id", stripePriceID),
											zap.String("subscription_id", func() string {
												if subscriptionID != "" {
													return subscriptionID
												}
												return "none"
											}()))

										if subscriptionID == "" {
											h.logger.Error("Cannot allocate credits from database: no subscription ID",
												zap.String("invoice_id", invoice.ID))
										} else {
											h.logger.Info("ATTEMPTING CREDIT ALLOCATION FROM DATABASE",
												zap.String("invoice_id", invoice.ID),
												zap.String("universal_id", universalID),
												zap.String("subscription_id", subscriptionID),
												zap.String("price_id", stripePriceID))

											if err := h.creditService.AllocateCreditsForPayment(
												c.Request().Context(),
												uuid.MustParse(universalID),
												invoice.ID,
												subscriptionID,
												stripePriceID,
											); err != nil {
												h.logger.Error("CREDIT ALLOCATION FROM DATABASE FAILED",
													zap.String("invoice_id", invoice.ID),
													zap.String("universal_id", universalID),
													zap.String("subscription_id", subscriptionID),
													zap.String("price_id", stripePriceID),
													zap.Error(err))
											} else {
												h.logger.Info("CREDIT ALLOCATION FROM DATABASE SUCCESSFUL",
													zap.String("invoice_id", invoice.ID),
													zap.String("universal_id", universalID),
													zap.String("subscription_id", subscriptionID),
													zap.String("price_id", stripePriceID))
											}
										}
									}
								} else {
									h.logger.Error("Cannot allocate credits: no price ID found")
								}
							} else {
								h.logger.Warn("First line item is not a map object")
							}
						} else {
							h.logger.Warn("No line items data found or empty")
						}
					} else {
						h.logger.Warn("No lines object found in raw invoice data")
					}
				} else {
					h.logger.Error("Failed to parse raw invoice for credit allocation",
						zap.String("invoice_id", invoice.ID),
						zap.Error(err))
				}
			} else {
				h.logger.Warn("Credit allocation skipped - preconditions not met",
					zap.Bool("has_credit_service", h.creditService != nil),
					zap.Bool("has_invoice_lines", invoice.Lines != nil),
					zap.Int("line_items_count", func() int {
						if invoice.Lines != nil {
							return len(invoice.Lines.Data)
						}
						return 0
					}()))
			}

			h.logger.Info("=== CREDIT ALLOCATION FLOW ANALYSIS END ===")

			// Update subscription period if applicable
			if invoice.Subscription != nil && invoice.Lines != nil && len(invoice.Lines.Data) > 0 {
				line := invoice.Lines.Data[0]
				if line.Period != nil && line.Period.End > 0 {
					// Update subscription in database with new period end
					h.logger.Info("Subscription period should be extended",
						zap.String("subscription_id", invoice.Subscription.ID),
						zap.Time("new_period_end", time.Unix(line.Period.End, 0)))
				}
			}
		}

	case stripe.EventTypeInvoicePaymentFailed:
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			h.logger.Error("Error parsing invoice", zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "Error parsing webhook"})
		}

		h.logger.Warn("PAYMENT FAILED",
			zap.String("invoice_id", invoice.ID),
			zap.Int64("amount_due", invoice.AmountDue),
		)

		h.mu.Lock()
		payment := PaymentData{
			InvoiceID: invoice.ID,
			Amount:    invoice.AmountDue,
			Status:    "failed",
			CreatedAt: time.Now(),
		}

		if invoice.Customer != nil {
			payment.CustomerID = invoice.Customer.ID
		}

		h.payments = append(h.payments, payment)
		h.mu.Unlock()

	case stripe.EventTypeProductCreated, stripe.EventTypeProductUpdated, stripe.EventTypeProductDeleted:
		if h.planSyncService != nil {
			if err := h.planSyncService.SyncProductEvent(c.Request().Context(), string(event.Type), event.Data.Raw); err != nil {
				h.logger.Error("Failed to sync product event",
					zap.String("event_type", string(event.Type)),
					zap.Error(err))
				return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to sync product"})
			}
			h.logger.Info("Product event synced successfully",
				zap.String("event_type", string(event.Type)))
		}

	case stripe.EventTypePriceCreated, stripe.EventTypePriceUpdated, stripe.EventTypePriceDeleted:
		if h.planSyncService != nil {
			if err := h.planSyncService.SyncPriceEvent(c.Request().Context(), string(event.Type), event.Data.Raw); err != nil {
				h.logger.Error("Failed to sync price event",
					zap.String("event_type", string(event.Type)),
					zap.Error(err))
				return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to sync price"})
			}
			h.logger.Info("Price event synced successfully",
				zap.String("event_type", string(event.Type)))
		}

	default:
		h.logger.Warn("Unhandled event type",
			zap.String("type", string(event.Type)),
		)
	}

	// Mark webhook as processed
	if h.webhookRepo != nil {
		if err := h.webhookRepo.MarkProcessed(c.Request().Context(), event.ID); err != nil {
			h.logger.Error("Failed to mark webhook as processed", zap.Error(err))
			// Continue even if marking fails
		}
	}

	return c.JSON(http.StatusOK, echo.Map{"received": true})
}

// isValidUUID checks if a string is a valid UUID
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// getMapKeys returns the keys of a map as a slice for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
