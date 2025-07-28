package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

type CheckoutHandler struct {
	logger              *zap.Logger
	clientURL           string
	customerMappingRepo repository.CustomerMappingRepository
}

func NewCheckoutHandler(logger *zap.Logger, clientURL string, customerMappingRepo repository.CustomerMappingRepository) *CheckoutHandler {
	return &CheckoutHandler{
		logger:              logger,
		clientURL:           clientURL,
		customerMappingRepo: customerMappingRepo,
	}
}

type CreateCheckoutRequest struct {
	PriceID string `json:"priceId" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	UserID  string `json:"userId" validate:"required"` // User ID from your auth system
}

type CreateCheckoutResponse struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Status      string `json:"status"`
	CheckoutURL string `json:"checkout_url"`
}

func (h *CheckoutHandler) CreateSubscription(c echo.Context) error {
	var req CreateCheckoutRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request body",
		})
	}

	// Validate user ID is a valid UUID
	if req.UserID == "" {
		h.logger.Error("User ID is required for subscription creation")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "User ID is required",
		})
	}

	if _, err := uuid.Parse(req.UserID); err != nil {
		h.logger.Error("Invalid user ID format",
			zap.String("user_id", req.UserID),
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "User ID must be a valid UUID",
			"details": "Expected format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		})
	}

	h.logger.Info("Creating subscription...",
		zap.String("price_id", req.PriceID),
		zap.String("email", req.Email),
		zap.String("user_id", req.UserID),
	)

	// Check if we already have a Stripe customer for this user
	var existingCustomerID string
	if h.customerMappingRepo != nil {
		existingMapping, err := h.customerMappingRepo.GetByUserID(c.Request().Context(), req.UserID)
		if err != nil {
			h.logger.Warn("Error checking for existing customer mapping",
				zap.String("user_id", req.UserID),
				zap.Error(err))
		} else if existingMapping != nil {
			existingCustomerID = existingMapping.StripeCustomerID
			h.logger.Info("Found existing Stripe customer",
				zap.String("customer_id", existingCustomerID),
				zap.String("user_id", req.UserID))
		}
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(h.clientURL + "/success.html"),
		CancelURL:  stripe.String(h.clientURL + "/cancel.html"),
		// Set metadata on subscription
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": req.UserID,
			},
		},
		// Session metadata
		Metadata: map[string]string{
			"user_id": req.UserID,
		},
	}

	// Use existing customer or create new one
	if existingCustomerID != "" {
		params.Customer = stripe.String(existingCustomerID)
		h.logger.Info("Using existing customer for checkout session",
			zap.String("customer_id", existingCustomerID))
	} else {
		params.CustomerEmail = stripe.String(req.Email)
		h.logger.Info("Creating new customer with email",
			zap.String("email", req.Email))
	}

	s, err := checkoutsession.New(params)
	if err != nil {
		h.logger.Error("Error creating subscription", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, CreateCheckoutResponse{
		ID:          s.ID,
		URL:         s.URL,
		Status:      "pending",
		CheckoutURL: s.URL,
	})
}

type CreatePortalRequest struct {
	CustomerID string `json:"customerId"`
}

func (h *CheckoutHandler) CreatePortalSession(c echo.Context) error {
	var req CreatePortalRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request body",
		})
	}

	h.logger.Info("Creating customer portal session...",
		zap.String("customer_id", req.CustomerID),
	)

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(req.CustomerID),
		ReturnURL: stripe.String(h.clientURL),
	}

	ps, err := portalsession.New(params)
	if err != nil {
		h.logger.Error("Error creating portal session", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	h.logger.Info("Portal Session Created",
		zap.String("portal_session_id", ps.ID),
		zap.String("portal_url", ps.URL),
		zap.Int64("created", ps.Created),
	)

	return c.JSON(http.StatusOK, echo.Map{
		"url": ps.URL,
	})
}
