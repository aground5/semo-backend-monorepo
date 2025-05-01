package controllers

import (
	"authn-server/internal/logics"
	"net/http"

	"github.com/labstack/echo/v4"
)

// CaptchaRequest is the payload for CAPTCHA generation requests
type CaptchaRequest struct {
	ForceCreate bool `json:"force_create" form:"force_create"` // Whether to force creation even if not required
}

// CaptchaVerifyRequest is the payload for CAPTCHA verification requests
type CaptchaVerifyRequest struct {
	ChallengeID string `json:"challenge_id" form:"challenge_id"` // The CAPTCHA challenge ID
	Response    string `json:"response" form:"response"`         // The user's response to the CAPTCHA
}

// GenerateCaptchaHandler creates a new CAPTCHA challenge
// POST /captcha/generate
func GenerateCaptchaHandler(c echo.Context) error {
	req := new(CaptchaRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Generate CAPTCHA challenge
	captchaResponse, err := logics.CaptchaSvc.GenerateCaptcha(c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, captchaResponse)
}

// VerifyCaptchaHandler verifies a CAPTCHA response
// POST /captcha/verify
func VerifyCaptchaHandler(c echo.Context) error {
	req := new(CaptchaVerifyRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.ChallengeID == "" || req.Response == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "challenge_id and response are required"})
	}

	// Verify CAPTCHA response
	verifyResponse, err := logics.CaptchaSvc.VerifyCaptcha(req.ChallengeID, req.Response, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, verifyResponse)
}

// CheckCaptchaRequiredHandler checks if CAPTCHA is required for the current IP or email
// GET /captcha/required
func CheckCaptchaRequiredHandler(c echo.Context) error {
	email := c.QueryParam("email")
	ip := c.RealIP()

	// Check if CAPTCHA is required
	required := logics.CaptchaSvc.IsCaptchaRequired(ip, email)

	return c.JSON(http.StatusOK, map[string]bool{"required": required})
}
