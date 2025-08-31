package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v79"
	portalsession "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/middleware/auth"
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
	PriceID string `json:"priceId"`
	Email   string `json:"email"`
	Mode    string `json:"mode"` // "embedded" or "" (기본값)
}

type CreateCheckoutResponse struct {
	ID           string `json:"id"`
	URL          string `json:"url,omitempty"`          // Hosted mode only
	CheckoutURL  string `json:"checkout_url,omitempty"` // Hosted mode only (legacy)
	ClientSecret string `json:"clientSecret,omitempty"` // Embedded mode only
	Status       string `json:"status"`
	SessionID    string `json:"sessionId"`
}

// Optional: Return URL 핸들러 추가 (필요한 경우)
func (h *CheckoutHandler) HandleReturn(c echo.Context) error {
	sessionID := c.QueryParam("session_id")

	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Session ID required",
		})
	}

	// Checkout session 조회
	s, err := checkoutsession.Get(sessionID, nil)
	if err != nil {
		h.logger.Error("Error retrieving checkout session",
			zap.String("session_id", sessionID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to retrieve session",
		})
	}

	// 결제 성공 여부 확인
	if s.Status == "complete" && s.PaymentStatus == "paid" {
		// 성공 페이지로 리다이렉트
		return c.Redirect(http.StatusFound, h.clientURL+"/?success=true")
	}

	// 실패 또는 취소된 경우
	return c.Redirect(http.StatusFound, h.clientURL+"/?canceled=true")
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

// CheckSessionStatus retrieves the status of a checkout session
func (h *CheckoutHandler) CheckSessionStatus(c echo.Context) error {
	// Get authenticated user from JWT
	user, err := auth.RequireAuth(c)
	if err != nil {
		return err // RequireAuth already returns the JSON error response
	}

	sessionID := c.Param("sessionId")

	h.logger.Info("Checking session status",
		zap.String("session_id", sessionID),
		zap.String("universal_id", user.UniversalID),
	)

	s, err := checkoutsession.Get(sessionID, nil)
	if err != nil {
		h.logger.Error("Failed to retrieve session",
			zap.String("session_id", sessionID),
			zap.String("universal_id", user.UniversalID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to retrieve session",
		})
	}

	// Validate user ownership of the session
	sessionUserID, exists := s.Metadata["user_id"]
	if !exists || sessionUserID != user.UniversalID {
		h.logger.Warn("Unauthorized session access attempt",
			zap.String("session_id", sessionID),
			zap.String("requesting_universal_id", user.UniversalID),
			zap.String("session_user_id", sessionUserID),
		)
		return c.JSON(http.StatusForbidden, echo.Map{
			"error": "Access denied: Session does not belong to authenticated user",
			"code":  "SESSION_ACCESS_DENIED",
		})
	}

	// Get customer ID - handle both ID and expandable object
	var customerID string
	if s.Customer != nil {
		customerID = s.Customer.ID
	}

	h.logger.Info("Session status retrieved",
		zap.String("session_id", sessionID),
		zap.String("status", string(s.Status)),
		zap.String("payment_status", string(s.PaymentStatus)),
		zap.String("customer_id", customerID),
		zap.String("universal_id", user.UniversalID),
	)

	return c.JSON(http.StatusOK, echo.Map{
		"status":        s.Status,
		"paymentStatus": s.PaymentStatus,
		"customerId":    customerID,
	})
}
