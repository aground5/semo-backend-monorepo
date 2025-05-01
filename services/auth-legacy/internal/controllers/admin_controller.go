package controllers

import (
	"authn-server/internal/logics"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// BlockIPRequest is the payload for blocking an IP address
type BlockIPRequest struct {
	IP        string `json:"ip" form:"ip"`
	Reason    string `json:"reason" form:"reason"`
	Duration  int    `json:"duration" form:"duration"` // Duration in hours, 0 means permanent
	Permanent bool   `json:"permanent" form:"permanent"`
}

// UnblockIPRequest is the payload for unblocking an IP address
type UnblockIPRequest struct {
	IP string `json:"ip" form:"ip"`
}

// ListBlockedIPsHandler returns a list of all blocked IPs
// GET /admin/blocked-ips
func ListBlockedIPsHandler(c echo.Context) error {
	var blockedIPs []models.BlockedIP

	if err := repositories.DBS.Postgres.Find(&blockedIPs).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error retrieving IP list"})
	}

	return c.JSON(http.StatusOK, blockedIPs)
}

// BlockIPHandler handles the endpoint to block a specific IP
// POST /admin/block-ip
func BlockIPHandler(c echo.Context) error {
	req := new(BlockIPRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.IP == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "IP address is required"})
	}

	// Check if there's an existing block record
	var existingBlock models.BlockedIP
	result := repositories.DBS.Postgres.Where("ip = ?", req.IP).First(&existingBlock)

	var blockedUntil time.Time
	if req.Permanent {
		// For permanent blocks, set expiry to 100 years in the future (effectively permanent)
		blockedUntil = time.Now().AddDate(100, 0, 0)
	} else {
		// Default to 24 hours or specified duration (in hours)
		duration := 24
		if req.Duration > 0 {
			duration = req.Duration
		}
		blockedUntil = time.Now().Add(time.Duration(duration) * time.Hour)
	}

	if result.Error == nil {
		// Update existing block record
		updates := map[string]interface{}{
			"reason":        req.Reason,
			"blocked_until": blockedUntil,
			"permanent":     req.Permanent,
		}

		if err := repositories.DBS.Postgres.Model(&existingBlock).Updates(updates).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error updating IP block"})
		}

		// Log the update with AuditLogService
		content := map[string]interface{}{
			"ip":            req.IP,
			"reason":        req.Reason,
			"blocked_until": blockedUntil,
			"permanent":     req.Permanent,
			"action":        "update",
		}
		logics.AuditLogSvc.AddLog(models.AuditLogTypeBlockedIPLogin, content, nil)

		return c.JSON(http.StatusOK, map[string]string{"message": "IP block updated"})
	} else {
		// Create new block record
		newBlock := models.BlockedIP{
			IP:           req.IP,
			Reason:       req.Reason,
			BlockedUntil: blockedUntil,
			Permanent:    req.Permanent,
		}

		if err := repositories.DBS.Postgres.Create(&newBlock).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error blocking IP"})
		}

		// Log the creation with AuditLogService
		content := map[string]interface{}{
			"ip":            req.IP,
			"reason":        req.Reason,
			"blocked_until": blockedUntil,
			"permanent":     req.Permanent,
			"action":        "create",
		}
		logics.AuditLogSvc.AddLog(models.AuditLogTypeBlockedIPLogin, content, nil)

		return c.JSON(http.StatusOK, map[string]string{"message": "IP blocked successfully"})
	}
}

// UnblockIPHandler handles the endpoint to unblock a specific IP
// POST /admin/unblock-ip
func UnblockIPHandler(c echo.Context) error {
	req := new(UnblockIPRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.IP == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "IP address is required"})
	}

	result := repositories.DBS.Postgres.Where("ip = ?", req.IP).Delete(&models.BlockedIP{})
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error unblocking IP"})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "No block record found for this IP"})
	}

	// Log the unblock with AuditLogService
	content := map[string]interface{}{
		"ip":     req.IP,
		"action": "unblock",
	}
	logics.AuditLogSvc.AddLog(models.AuditLogTypeBlockedIPLogin, content, nil)

	return c.JSON(http.StatusOK, map[string]string{"message": "IP unblocked successfully"})
}
