package logics

import (
	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DeviceInfo contains information about a device
type DeviceInfo struct {
	DeviceUID  string // Device unique identifier
	DeviceName string // User-defined device name
	DeviceType string // Device type (mobile, tablet, desktop)
	UserAgent  string // User-Agent string
	IP         string // IP address
}

// TrustedDeviceService provides trusted device management functionality
type TrustedDeviceService struct{}

// NewTrustedDeviceService creates a new TrustedDeviceService
func NewTrustedDeviceService() *TrustedDeviceService {
	return &TrustedDeviceService{}
}

// GetUserTrustedDevices returns a user's trusted devices
func (s *TrustedDeviceService) GetUserTrustedDevices(userID string) ([]models.TrustedDevice, error) {
	var devices []models.TrustedDevice

	// Only retrieve non-expired trusted devices
	err := repositories.DBS.Postgres.
		Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Find(&devices).Error

	return devices, err
}

// AddTrustedDevice adds a trusted device
func (s *TrustedDeviceService) AddTrustedDevice(userID string, deviceInfo DeviceInfo) (*models.TrustedDevice, error) {
	// Check for existing device
	var existing models.TrustedDevice
	result := repositories.DBS.Postgres.
		Where("user_id = ? AND device_uid = ?", userID, deviceInfo.DeviceUID).
		First(&existing)

	// Detect device type
	deviceType := s.detectDeviceType(deviceInfo.UserAgent)
	if deviceInfo.DeviceType != "" {
		deviceType = deviceInfo.DeviceType
	}

	// Generate device name if not provided
	deviceName := deviceInfo.DeviceName
	if deviceName == "" {
		deviceName = s.generateDeviceName(deviceType)
	}

	// Set expiration (default 90 days)
	expiresAt := time.Now().AddDate(0, 3, 0)

	if result.Error == nil {
		// Update existing device
		updates := map[string]interface{}{
			"device_name": deviceName,
			"device_type": deviceType,
			"user_agent":  deviceInfo.UserAgent,
			"last_ip":     deviceInfo.IP,
			"last_used":   time.Now(),
			"expires_at":  expiresAt,
		}

		if err := repositories.DBS.Postgres.Model(&existing).Updates(updates).Error; err != nil {
			return nil, err
		}

		return &existing, nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// Add new device
	device := models.TrustedDevice{
		UserID:     userID,
		DeviceUID:  deviceInfo.DeviceUID,
		DeviceName: deviceName,
		DeviceType: deviceType,
		UserAgent:  deviceInfo.UserAgent,
		LastIP:     deviceInfo.IP,
		LastUsed:   time.Now(),
		ExpiresAt:  expiresAt,
	}

	if err := repositories.DBS.Postgres.Create(&device).Error; err != nil {
		return nil, err
	}

	// Create notification (new device registered)
	notificationData := map[string]interface{}{
		"device_name": deviceName,
		"device_type": deviceType,
		"ip":          deviceInfo.IP,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	NotificationSvc.CreateSecurityAlert(
		userID,
		models.NotificationTypeNewDevice,
		"새 장치가 등록되었습니다",
		fmt.Sprintf("%s에서 새 장치(%s)가 신뢰할 수 있는 장치로 등록되었습니다.", deviceInfo.IP, deviceName),
		notificationData,
	)

	// Update any unknown device alerts
	repositories.DBS.Postgres.Model(&models.UnknownDeviceAlert{}).
		Where("user_id = ? AND device_uid = ? AND confirmed_by = ?", userID, deviceInfo.DeviceUID, "none").
		Updates(map[string]interface{}{
			"confirmed_by": "user",
			"action":       "allowed",
		})

	return &device, nil
}

// RemoveTrustedDevice removes a trusted device
func (s *TrustedDeviceService) RemoveTrustedDevice(userID, deviceUID string) error {
	result := repositories.DBS.Postgres.
		Where("user_id = ? AND device_uid = ?", userID, deviceUID).
		Delete(&models.TrustedDevice{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("device not found")
	}

	return nil
}

// IsTrustedDevice checks if a device is trusted
func (s *TrustedDeviceService) IsTrustedDevice(userID, deviceUID string) (bool, error) {
	var count int64

	err := repositories.DBS.Postgres.Model(&models.TrustedDevice{}).
		Where("user_id = ? AND device_uid = ? AND expires_at > ?", userID, deviceUID, time.Now()).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// DetectUnknownDevice detects unknown devices and creates alerts if necessary
func (s *TrustedDeviceService) DetectUnknownDevice(userID string, deviceUUID uuid.UUID, ip, userAgent, location string) (bool, error) {
	deviceUID := deviceUUID.String()

	// 1. Check if this is a trusted device
	isTrusted, err := s.IsTrustedDevice(userID, deviceUID)
	if err != nil {
		return false, err
	}

	if isTrusted {
		// Update last used time for trusted device
		repositories.DBS.Postgres.Model(&models.TrustedDevice{}).
			Where("user_id = ? AND device_uid = ?", userID, deviceUID).
			Updates(map[string]interface{}{
				"last_used": time.Now(),
				"last_ip":   ip,
			})
		return false, nil
	}

	// 2. Check for existing alert for this device
	var existingAlert models.UnknownDeviceAlert
	result := repositories.DBS.Postgres.
		Where("user_id = ? AND device_uid = ? AND created_at > ?", userID, deviceUID, time.Now().AddDate(0, 0, -7)).
		First(&existingAlert)

	if result.Error == nil {
		// Alert already exists, update if needed
		if !existingAlert.AlertSent {
			repositories.DBS.Postgres.Model(&existingAlert).
				Update("alert_sent", true)
		}
		return true, nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, result.Error
	}

	// 3. Calculate risk score (0-100)
	riskScore := s.calculateDeviceRiskScore(userID, deviceUID, ip, userAgent)

	// 4. Create unknown device alert
	alert := models.UnknownDeviceAlert{
		UserID:      userID,
		DeviceUID:   deviceUID,
		IP:          ip,
		UserAgent:   userAgent,
		Location:    location,
		RiskScore:   riskScore,
		AlertSent:   false,
		ConfirmedBy: "none",
		Action:      "pending",
	}

	if err := repositories.DBS.Postgres.Create(&alert).Error; err != nil {
		return false, err
	}

	// 5. Send alert for high-risk devices
	if riskScore >= 50 {
		// Send alert asynchronously
		go func() {
			err := NotificationSvc.CreateUnknownDeviceAlert(userID, deviceUID, ip, userAgent, location, riskScore)
			if err != nil {
				configs.Logger.Error("Failed to send unknown device alert", zap.Error(err))
			} else {
				// Update alert sent status on success
				repositories.DBS.Postgres.Model(&alert).
					Update("alert_sent", true)
			}
		}()
	}

	return true, nil
}

// ConfirmDevice confirms and handles an unknown device
func (s *TrustedDeviceService) ConfirmDevice(alertID uint, userID string, isTrusted bool, confirmedBy string) error {
	// 1. Find the alert
	var alert models.UnknownDeviceAlert
	if err := repositories.DBS.Postgres.Where("id = ? AND user_id = ?", alertID, userID).First(&alert).Error; err != nil {
		return err
	}

	// 2. Check if already processed
	if alert.ConfirmedBy != "none" {
		return errors.New("alert already processed")
	}

	// 3. Update the alert
	action := "blocked"
	if isTrusted {
		action = "allowed"
	}

	updates := map[string]interface{}{
		"confirmed_by": confirmedBy,
		"action":       action,
	}

	if err := repositories.DBS.Postgres.Model(&alert).Updates(updates).Error; err != nil {
		return err
	}

	// 4. Add as trusted device if trusted
	if isTrusted {
		deviceInfo := DeviceInfo{
			DeviceUID:  alert.DeviceUID,
			DeviceName: "", // Auto-generate
			UserAgent:  alert.UserAgent,
			IP:         alert.IP,
		}

		if _, err := s.AddTrustedDevice(userID, deviceInfo); err != nil {
			return err
		}
	} else {
		// If not trusted, terminate existing sessions
		// TODO: Implement session termination for this device
	}

	return nil
}

// Private helper methods

// detectDeviceType detects device type from User-Agent
func (s *TrustedDeviceService) detectDeviceType(userAgent string) string {
	// Simple implementation: determine device type from User-Agent
	userAgent = strings.ToLower(userAgent)

	if strings.Contains(userAgent, "mobile") || strings.Contains(userAgent, "android") || strings.Contains(userAgent, "iphone") {
		return "mobile"
	} else if strings.Contains(userAgent, "tablet") || strings.Contains(userAgent, "ipad") {
		return "tablet"
	} else {
		return "desktop"
	}
}

// generateDeviceName generates a default name for a device type
func (s *TrustedDeviceService) generateDeviceName(deviceType string) string {
	switch deviceType {
	case "mobile":
		return "모바일 기기"
	case "tablet":
		return "태블릿"
	default:
		return "데스크톱"
	}
}

// calculateDeviceRiskScore calculates a risk score for an unknown device
func (s *TrustedDeviceService) calculateDeviceRiskScore(userID, deviceUID, ip, userAgent string) int {
	score := 50 // Base score

	// 1. Analyze previous login attempts
	var failedLoginCount int64
	repositories.DBS.Postgres.Model(&models.LoginAttempt{}).
		Where("user_id = ? AND success = ? AND device_uid = ? AND created_at > ?",
			userID, false, deviceUID, time.Now().AddDate(0, 0, -1)).
		Count(&failedLoginCount)

	if failedLoginCount > 0 {
		score += int(failedLoginCount) * 5 // +5 points per failed login
	}

	// 2. Check IP risk
	var blockedIPCount int64
	repositories.DBS.Postgres.Model(&models.BlockedIP{}).
		Where("ip = ?", ip).
		Count(&blockedIPCount)

	if blockedIPCount > 0 {
		score += 20 // +20 points for blocked IP
	}

	// 3. Analyze User-Agent
	if len(userAgent) < 20 || len(userAgent) > 500 {
		score += 10 // Abnormal User-Agent length
	}

	// Check for bot signatures
	botSignatures := []string{"bot", "crawler", "spider", "curl", "wget", "python"}
	for _, sig := range botSignatures {
		if strings.Contains(strings.ToLower(userAgent), sig) {
			score += 15
			break
		}
	}

	// Limit score range (0-100)
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	return score
}

// Global instance of TrustedDeviceService
var TrustedDeviceSvc = NewTrustedDeviceService()
