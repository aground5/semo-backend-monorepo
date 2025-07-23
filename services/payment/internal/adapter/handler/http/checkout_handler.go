package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"go.uber.org/zap"
)

type CheckoutHandler struct {
	logger    *zap.Logger
	clientURL string
}

func NewCheckoutHandler(logger *zap.Logger, clientURL string) *CheckoutHandler {
	return &CheckoutHandler{
		logger:    logger,
		clientURL: clientURL,
	}
}

type CreateCheckoutRequest struct {
	PriceID string `json:"priceId" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
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
	
	h.logger.Info("Creating subscription...",
		zap.String("price_id", req.PriceID),
		zap.String("email", req.Email),
	)
	
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:          stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:    stripe.String(h.clientURL + "/success.html"),
		CancelURL:     stripe.String(h.clientURL + "/cancel.html"),
		CustomerEmail: stripe.String(req.Email),
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