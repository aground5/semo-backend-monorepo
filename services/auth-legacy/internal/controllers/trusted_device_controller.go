package controllers

import (
	"authn-server/internal/logics"
	"authn-server/internal/middlewares"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// AddTrustedDeviceRequest is the payload for adding a trusted device
type AddTrustedDeviceRequest struct {
	DeviceUID  string `json:"device_uid" form:"device_uid"`   // Device unique identifier
	DeviceName string `json:"device_name" form:"device_name"` // User-defined device name
	DeviceType string `json:"device_type" form:"device_type"` // Device type (optional)
}

// RemoveTrustedDeviceRequest is the payload for removing a trusted device
type RemoveTrustedDeviceRequest struct {
	DeviceUID string `json:"device_uid" form:"device_uid"` // Device unique identifier
}

// ConfirmDeviceRequest is the payload for confirming an unknown device
type ConfirmDeviceRequest struct {
	AlertID   uint `json:"alert_id" form:"alert_id"`     // Unknown device alert ID
	IsTrusted bool `json:"is_trusted" form:"is_trusted"` // Whether to trust the device
}

// GetTrustedDevicesHandler returns a user's trusted devices
// GET /trusted-devices
func GetTrustedDevicesHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Get trusted devices
	devices, err := logics.TrustedDeviceSvc.GetUserTrustedDevices(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, devices)
}

// AddTrustedDeviceHandler adds a device to a user's trusted devices
// POST /trusted-devices
func AddTrustedDeviceHandler(c echo.Context) error {
	req := new(AddTrustedDeviceRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.DeviceUID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_uid is required"})
	}

	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Create device info
	deviceInfo := logics.DeviceInfo{
		DeviceUID:  req.DeviceUID,
		DeviceName: req.DeviceName,
		DeviceType: req.DeviceType,
		UserAgent:  c.Request().UserAgent(),
		IP:         c.RealIP(),
	}

	// Add trusted device
	device, err := logics.TrustedDeviceSvc.AddTrustedDevice(userID, deviceInfo)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Log the addition
	content := map[string]interface{}{
		"device_uid":  req.DeviceUID,
		"device_name": req.DeviceName,
		"device_type": device.DeviceType,
		"ip":          c.RealIP(),
	}
	logics.AuditLogSvc.AddLog(models.AuditLogTypeNewDeviceRegistered, content, &userID)

	return c.JSON(http.StatusOK, device)
}

// RemoveTrustedDeviceHandler removes a device from a user's trusted devices
// DELETE /trusted-devices
func RemoveTrustedDeviceHandler(c echo.Context) error {
	req := new(RemoveTrustedDeviceRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.DeviceUID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_uid is required"})
	}

	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Remove trusted device
	if err := logics.TrustedDeviceSvc.RemoveTrustedDevice(userID, req.DeviceUID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Log the removal
	content := map[string]interface{}{
		"device_uid": req.DeviceUID,
		"action":     "remove",
	}
	logics.AuditLogSvc.AddLog(models.AuditLogTypeNewDeviceRegistered, content, &userID)

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// CheckDeviceTrustedHandler checks if a device is trusted
// GET /trusted-devices/check
func CheckDeviceTrustedHandler(c echo.Context) error {
	deviceUID := c.QueryParam("device_uid")
	if deviceUID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_uid is required"})
	}

	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Check if device is trusted
	isTrusted, err := logics.TrustedDeviceSvc.IsTrustedDevice(userID, deviceUID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]bool{"trusted": isTrusted})
}

// GetUnknownDeviceAlertsHandler returns a user's unknown device alerts
// GET /trusted-devices/alerts
func GetUnknownDeviceAlertsHandler(c echo.Context) error {
	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Get unknown device alerts
	var alerts []models.UnknownDeviceAlert
	if err := repositories.DBS.Postgres.Where("user_id = ?", userID).
		Order("created_at DESC").Find(&alerts).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, alerts)
}

// ConfirmDeviceHandler confirms an unknown device
// POST /trusted-devices/confirm
func ConfirmDeviceHandler(c echo.Context) error {
	req := new(ConfirmDeviceRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.AlertID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "alert_id is required"})
	}

	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Confirm device
	if err := logics.TrustedDeviceSvc.ConfirmDevice(req.AlertID, userID, req.IsTrusted, "user"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Log the confirmation
	content := map[string]interface{}{
		"alert_id":   req.AlertID,
		"is_trusted": req.IsTrusted,
		"action":     "confirm",
	}
	logics.AuditLogSvc.AddLog(models.AuditLogTypeSecurityAlert, content, &userID)

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// DetectUnknownDeviceHandler checks if a device is unknown and creates alerts if necessary
// POST /trusted-devices/detect
func DetectUnknownDeviceHandler(c echo.Context) error {
	deviceUIDStr := c.QueryParam("device_uid")
	if deviceUIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_uid is required"})
	}

	// Get user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Parse UUID
	deviceUID, err := uuid.Parse(deviceUIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid device_uid"})
	}

	// Use geolocation service to get location info (placeholder)
	location := "Unknown Location"

	// Detect unknown device
	isUnknown, err := logics.TrustedDeviceSvc.DetectUnknownDevice(userID, deviceUID, c.RealIP(), c.Request().UserAgent(), location)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]bool{"unknown_device": isUnknown})
}
