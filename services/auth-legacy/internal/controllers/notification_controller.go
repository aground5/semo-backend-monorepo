package controllers

import (
	"authn-server/internal/logics"
	"authn-server/internal/middlewares"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// NotificationPreferenceRequest is the payload for updating notification preferences
type NotificationPreferenceRequest struct {
	EmailEnabled   bool `json:"email_enabled" form:"email_enabled"`     // Whether to enable email notifications
	SMSEnabled     bool `json:"sms_enabled" form:"sms_enabled"`         // Whether to enable SMS notifications
	PushEnabled    bool `json:"push_enabled" form:"push_enabled"`       // Whether to enable push notifications
	InAppEnabled   bool `json:"in_app_enabled" form:"in_app_enabled"`   // Whether to enable in-app notifications
	SecurityAlerts bool `json:"security_alerts" form:"security_alerts"` // Whether to receive security alerts
	LoginAlerts    bool `json:"login_alerts" form:"login_alerts"`       // Whether to receive login alerts
}

// MarkReadRequest is the payload for marking a notification as read
type MarkReadRequest struct {
	NotificationID uint `json:"notification_id" form:"notification_id"` // The notification ID to mark as read
}

// GetNotificationsHandler returns a user's notifications
// GET /notifications
func GetNotificationsHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Check whether to include read notifications
	includeRead := c.QueryParam("include_read") == "true"

	// Get user's notifications
	notifications, err := logics.NotificationSvc.GetUserNotifications(userID, includeRead)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, notifications)
}

// MarkNotificationAsReadHandler marks a notification as read
// POST /notifications/mark-read
func MarkNotificationAsReadHandler(c echo.Context) error {
	req := new(MarkReadRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.NotificationID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "notification_id is required"})
	}

	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Mark notification as read
	if err := logics.NotificationSvc.MarkNotificationAsRead(req.NotificationID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// GetNotificationPreferencesHandler returns a user's notification preferences
// GET /notifications/preferences
func GetNotificationPreferencesHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Get user's notification preferences
	preferences, err := logics.NotificationSvc.GetUserNotificationPreferences(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, preferences)
}

// UpdateNotificationPreferencesHandler updates a user's notification preferences
// PUT /notifications/preferences
func UpdateNotificationPreferencesHandler(c echo.Context) error {
	req := new(NotificationPreferenceRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Create preferences object
	preferences := &models.NotificationPreference{
		UserID:         userID,
		EmailEnabled:   req.EmailEnabled,
		SMSEnabled:     req.SMSEnabled,
		PushEnabled:    req.PushEnabled,
		InAppEnabled:   req.InAppEnabled,
		SecurityAlerts: req.SecurityAlerts,
		LoginAlerts:    req.LoginAlerts,
	}

	// Update notification preferences
	if err := logics.NotificationSvc.UpdateUserNotificationPreferences(userID, preferences); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, preferences)
}

// MarkAllNotificationsAsReadHandler marks all of a user's notifications as read
// POST /notifications/mark-all-read
func MarkAllNotificationsAsReadHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Mark all notifications as read
	now := time.Now()
	result := repositories.DBS.Postgres.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		})

	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": result.Error.Error()})
	}

	return c.JSON(http.StatusOK, map[string]int64{"updated_count": result.RowsAffected})
}

// GetUnreadNotificationCountHandler returns the count of a user's unread notifications
// GET /notifications/unread-count
func GetUnreadNotificationCountHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Count unread notifications
	var count int64
	if err := repositories.DBS.Postgres.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]int64{"unread_count": count})
}
