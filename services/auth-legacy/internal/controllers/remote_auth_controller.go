package controllers

import (
	"authn-server/internal/logics"
	"authn-server/internal/middlewares"
	"authn-server/internal/models"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// ActivityResponse is the response format for activity information
type ActivityResponse struct {
	SessionID    string `json:"session_id"`
	UserID       string `json:"user_id"`
	TokenGroupID uint   `json:"token_group_id,omitempty"`
	IP           string `json:"ip"`
	UserAgent    string `json:"user_agent"`
	DeviceUID    string `json:"device_uid,omitempty"`
	LoginAt      string `json:"login_at"`
	Location     string `json:"location,omitempty"`
}

// DeactivateActivityRequest is the payload for activity deactivation
type DeactivateActivityRequest struct {
	SessionID string `json:"session_id" form:"session_id"` // The session ID to deactivate
}

// ListActiveActivitiesHandler returns active sessions for the authenticated user
// GET /remote-auth/activities
func ListActiveActivitiesHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Get active activities
	activities, err := logics.RemoteAuthSvc.GetActiveActivities(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Format response
	var responses []ActivityResponse
	for _, act := range activities {
		deviceUID := ""
		if act.DeviceUID != nil {
			deviceUID = act.DeviceUID.String()
		}

		responses = append(responses, ActivityResponse{
			SessionID:    act.SessionID,
			UserID:       act.UserID,
			TokenGroupID: act.TokenGroupID,
			IP:           act.IP,
			UserAgent:    act.UserAgent,
			DeviceUID:    deviceUID,
			LoginAt:      act.LoginAt.Format(time.RFC3339),
			Location:     act.LocationInfo,
		})
	}

	if responses == nil {
		responses = []ActivityResponse{}
	}

	return c.JSON(http.StatusOK, responses)
}

// DeactivateActivityHandler forcibly logs out a session
// POST /remote-auth/deactivate
func DeactivateActivityHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	req := new(DeactivateActivityRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.SessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	// Deactivate the activity
	if err := logics.RemoteAuthSvc.DeactivateActivity(req.SessionID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Log the deactivation
	content := map[string]interface{}{
		"session_id": req.SessionID,
		"user_id":    userID,
		"ip":         c.RealIP(),
	}
	logics.AuditLogSvc.AddLog(models.AuditLogTypeLogoutSuccess, content, &userID)

	return c.JSON(http.StatusOK, map[string]string{"message": "Activity deactivated successfully"})
}
