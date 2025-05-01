package controllers

import (
	"authn-server/internal/logics"
	"authn-server/internal/middlewares"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"net/http"

	"github.com/labstack/echo/v4"
)

// TwoFactorSetupResponse contains information for setting up 2FA
type TwoFactorSetupResponse struct {
	Secret            string `json:"secret"`              // TOTP secret key
	QRCodeURL         string `json:"qr_code_url"`         // QR code URL for scanning
	RecoveryCodeCount int    `json:"recovery_code_count"` // Number of recovery codes provided
}

// TwoFactorVerifyRequest is the payload for verifying a 2FA code
type TwoFactorVerifyRequest struct {
	Code string `json:"code" form:"code"` // The verification code
}

// TwoFactorChallengeRequest is the payload for completing a 2FA challenge
type TwoFactorChallengeRequest struct {
	ChallengeID string `json:"challenge_id" form:"challenge_id"` // Challenge identifier
	Code        string `json:"code" form:"code"`                 // The verification code
}

// SetupTwoFactorHandler starts the 2FA setup process
// POST /two-factor/setup
func SetupTwoFactorHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Get user email for 2FA setup
	var user models.User
	if err := repositories.DBS.Postgres.Select("email").Where("id = ?", userID).First(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "User not found"})
	}

	// Generate 2FA secret
	key, err := logics.TwoFactorSvc.GenerateSecret(userID, user.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Create response
	response := TwoFactorSetupResponse{
		Secret:            key.Secret(),
		QRCodeURL:         key.URL(),
		RecoveryCodeCount: 8, // Number of backup codes
	}

	// Log the setup action
	content := map[string]interface{}{
		"action": "setup_started",
	}
	logics.AuditLogSvc.AddLog(models.AuditLogType2FAEnabled, content, &userID)

	return c.JSON(http.StatusOK, response)
}

// VerifyAndEnableTwoFactorHandler verifies a 2FA code and enables 2FA
// POST /two-factor/verify
func VerifyAndEnableTwoFactorHandler(c echo.Context) error {
	req := new(TwoFactorVerifyRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Code is required"})
	}

	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Verify code and enable 2FA
	valid, backupCodes, err := logics.TwoFactorSvc.VerifyAndEnable(userID, req.Code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !valid {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid code"})
	}

	// Log the 2FA enablement
	content := map[string]interface{}{
		"action": "enabled",
		"method": "totp",
	}
	logics.AuditLogSvc.AddLog(models.AuditLogType2FAEnabled, content, &userID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":      true,
		"backup_codes": backupCodes,
	})
}

// DisableTwoFactorHandler disables 2FA for a user
// POST /two-factor/disable
func DisableTwoFactorHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Disable 2FA
	if err := logics.TwoFactorSvc.Disable2FA(userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Log the 2FA disablement
	content := map[string]interface{}{
		"action": "disabled",
	}
	logics.AuditLogSvc.AddLog(models.AuditLogType2FADisabled, content, &userID)

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// CheckTwoFactorEnabledHandler checks if 2FA is enabled for a user
// GET /two-factor/status
func CheckTwoFactorEnabledHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Check if 2FA is enabled
	enabled, err := logics.TwoFactorSvc.Is2FAEnabled(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]bool{"enabled": enabled})
}

// CreateTwoFactorChallengeHandler creates a 2FA challenge
// POST /two-factor/challenge
func CreateTwoFactorChallengeHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Create 2FA challenge
	challengeID, err := logics.TwoFactorSvc.CreateChallenge(userID, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Log the challenge creation
	content := map[string]interface{}{
		"action":       "challenge_created",
		"challenge_id": challengeID,
		"ip":           c.RealIP(),
	}
	logics.AuditLogSvc.AddLog(models.AuditLogType2FAVerified, content, &userID)

	return c.JSON(http.StatusOK, map[string]string{"challenge_id": challengeID})
}

// CompleteTwoFactorChallengeHandler completes a 2FA challenge
// POST /two-factor/complete-challenge
func CompleteTwoFactorChallengeHandler(c echo.Context) error {
	req := new(TwoFactorChallengeRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.ChallengeID == "" || req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "challenge_id and code are required"})
	}

	// Complete 2FA challenge
	valid, err := logics.TwoFactorSvc.CompleteChallenge(req.ChallengeID, req.Code, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !valid {
		// Log failed verification attempt
		var challenge models.TwoFactorChallenge
		if err := repositories.DBS.Postgres.Where("challenge_id = ?", req.ChallengeID).First(&challenge).Error; err == nil {
			content := map[string]interface{}{
				"action":       "verification_failed",
				"challenge_id": req.ChallengeID,
				"ip":           c.RealIP(),
			}
			logics.AuditLogSvc.AddLog(models.AuditLogType2FAVerified, content, &challenge.UserID)
		}

		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid code"})
	}

	// Get challenge information for logging
	var challenge models.TwoFactorChallenge
	if err := repositories.DBS.Postgres.Where("challenge_id = ?", req.ChallengeID).First(&challenge).Error; err == nil {
		content := map[string]interface{}{
			"action":       "verification_succeeded",
			"challenge_id": req.ChallengeID,
			"ip":           c.RealIP(),
		}
		logics.AuditLogSvc.AddLog(models.AuditLogType2FAVerified, content, &challenge.UserID)
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}
