package logics

import (
	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// NotificationService provides notification-related functionality
type NotificationService struct{}

// NewNotificationService creates a new NotificationService
func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(userID *string, notificationType models.NotificationType,
	channel models.NotificationChannel, title, content string, data map[string]interface{}) (*models.Notification, error) {

	// 1. Serialize JSON data
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// 2. Create notification record
	notification := &models.Notification{
		UserID:  userID,
		Type:    notificationType,
		Channel: channel,
		Title:   title,
		Content: content,
		Data:    dataJSON,
		Read:    false,
	}

	if err := repositories.DBS.Postgres.Create(notification).Error; err != nil {
		return nil, err
	}

	// 3. Send notification (asynchronously)
	go s.sendNotification(notification)

	return notification, nil
}

// GetUserNotifications retrieves a user's notifications
func (s *NotificationService) GetUserNotifications(userID string, includeRead bool) ([]models.Notification, error) {
	var notifications []models.Notification
	query := repositories.DBS.Postgres.Where("user_id = ?", userID)

	if !includeRead {
		query = query.Where("read = ?", false)
	}

	err := query.Order("created_at DESC").Find(&notifications).Error
	return notifications, err
}

// MarkNotificationAsRead marks a notification as read
func (s *NotificationService) MarkNotificationAsRead(notificationID uint, userID string) error {
	now := time.Now()
	result := repositories.DBS.Postgres.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("notification not found or not owned by user")
	}

	return nil
}

// GetUserNotificationPreferences retrieves a user's notification preferences
func (s *NotificationService) GetUserNotificationPreferences(userID string) (*models.NotificationPreference, error) {
	var preferences models.NotificationPreference
	err := repositories.DBS.Postgres.Where("user_id = ?", userID).First(&preferences).Error

	// Create default preferences if none exist
	if err != nil {
		preferences = models.NotificationPreference{
			UserID:         userID,
			EmailEnabled:   true,
			SMSEnabled:     false,
			PushEnabled:    true,
			InAppEnabled:   true,
			SecurityAlerts: true,
			LoginAlerts:    true,
		}

		if err := repositories.DBS.Postgres.Create(&preferences).Error; err != nil {
			return nil, err
		}
	}

	return &preferences, nil
}

// UpdateUserNotificationPreferences updates a user's notification preferences
func (s *NotificationService) UpdateUserNotificationPreferences(userID string,
	preferences *models.NotificationPreference) error {

	// Verify ownership
	if userID != preferences.UserID {
		return fmt.Errorf("user ID mismatch")
	}

	// Check for existing preferences
	var existingPrefs models.NotificationPreference
	result := repositories.DBS.Postgres.Where("user_id = ?", userID).First(&existingPrefs)

	if result.Error != nil {
		// Create new preferences if none exist
		return repositories.DBS.Postgres.Create(preferences).Error
	}

	// Set ID and update
	preferences.ID = existingPrefs.ID
	return repositories.DBS.Postgres.Save(preferences).Error
}

// CreateSecurityAlert creates a security alert notification
func (s *NotificationService) CreateSecurityAlert(userID string, alertType models.NotificationType,
	title, content string, data map[string]interface{}) error {

	// 1. Get user notification preferences
	prefs, err := s.GetUserNotificationPreferences(userID)
	if err != nil {
		configs.Logger.Error("Failed to get notification preferences",
			zap.Error(err), zap.String("userID", userID))
		// Use default preferences if we can't get user preferences
		prefs = &models.NotificationPreference{
			UserID:         userID,
			EmailEnabled:   true,
			SecurityAlerts: true,
		}
	}

	// 2. Check if security alerts are disabled
	if !prefs.SecurityAlerts {
		configs.Logger.Info("Security alerts disabled for user", zap.String("userID", userID))
		return nil
	}

	// 3. Send notifications on enabled channels
	if prefs.EmailEnabled {
		_, err := s.CreateNotification(&userID, alertType, models.NotificationChannelEmail, title, content, data)
		if err != nil {
			configs.Logger.Error("Failed to create email notification", zap.Error(err))
		}
	}

	if prefs.SMSEnabled {
		_, err := s.CreateNotification(&userID, alertType, models.NotificationChannelSMS, title, content, data)
		if err != nil {
			configs.Logger.Error("Failed to create SMS notification", zap.Error(err))
		}
	}

	// Always create in-app notification
	_, err = s.CreateNotification(&userID, alertType, models.NotificationChannelInApp, title, content, data)
	if err != nil {
		configs.Logger.Error("Failed to create in-app notification", zap.Error(err))
		return err
	}

	return nil
}

// CreateUnknownDeviceAlert creates an alert for unknown device login
func (s *NotificationService) CreateUnknownDeviceAlert(userID, deviceUID, ip, userAgent, location string, riskScore int) error {
	// 1. Set up alert data
	data := map[string]interface{}{
		"device_uid": deviceUID,
		"ip":         ip,
		"user_agent": userAgent,
		"location":   location,
		"risk_score": riskScore,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// 2. Save alert record in database
	alert := &models.UnknownDeviceAlert{
		UserID:      userID,
		DeviceUID:   deviceUID,
		IP:          ip,
		UserAgent:   userAgent,
		Location:    location,
		RiskScore:   riskScore,
		AlertSent:   true,
		ConfirmedBy: "none",
		Action:      "pending",
	}

	if err := repositories.DBS.Postgres.Create(alert).Error; err != nil {
		configs.Logger.Error("Failed to create unknown device alert", zap.Error(err))
	}

	// 3. Send alert to user
	err := s.CreateSecurityAlert(
		userID,
		models.NotificationTypeUnknownDevice,
		"알 수 없는 장치에서 로그인 시도",
		fmt.Sprintf("위치: %s에서 알 수 없는 장치로 계정 로그인이 감지되었습니다. 본인이 아닌 경우 즉시 비밀번호를 변경하시기 바랍니다.", location),
		data,
	)

	return err
}

// Private helper methods

// sendNotification handles the actual sending of notifications
func (s *NotificationService) sendNotification(notification *models.Notification) {
	switch notification.Channel {
	case models.NotificationChannelEmail:
		s.sendEmailNotification(notification)
	case models.NotificationChannelSMS:
		s.sendSMSNotification(notification)
	case models.NotificationChannelPush:
		s.sendPushNotification(notification)
	case models.NotificationChannelInApp:
		// In-app notifications are automatically stored in DB, no further action needed
	case models.NotificationChannelAdmin:
		s.sendAdminNotification(notification)
	}

	// Update sent time
	now := time.Now()
	notification.SentAt = &now
	repositories.DBS.Postgres.Model(notification).UpdateColumn("sent_at", now)
}

// sendEmailNotification sends an email notification
func (s *NotificationService) sendEmailNotification(notification *models.Notification) {
	// Get email address
	if notification.UserID == nil {
		configs.Logger.Error("Cannot send email notification: user ID is nil")
		return
	}

	var user models.User
	if err := repositories.DBS.Postgres.Where("id = ?", *notification.UserID).First(&user).Error; err != nil {
		configs.Logger.Error("Failed to find user for email notification",
			zap.Error(err), zap.String("userID", *notification.UserID))
		return
	}

	// Send email
	err := EmailSvc.SendEmail(
		configs.Configs.Email.SenderEmail,
		user.Email,
		notification.Title,
		fmt.Sprintf("<h1>%s</h1><p>%s</p>", notification.Title, notification.Content),
	)

	if err != nil {
		configs.Logger.Error("Failed to send email notification",
			zap.Error(err), zap.String("userID", *notification.UserID))
	} else {
		configs.Logger.Info("Email notification sent",
			zap.String("userID", *notification.UserID),
			zap.String("title", notification.Title))
	}
}

// sendSMSNotification sends an SMS notification (placeholder)
func (s *NotificationService) sendSMSNotification(notification *models.Notification) {
	// TODO: Implement SMS service
	configs.Logger.Info("SMS notification would be sent",
		zap.Any("notification", notification))
}

// sendPushNotification sends a push notification (placeholder)
func (s *NotificationService) sendPushNotification(notification *models.Notification) {
	// TODO: Implement push notification service
	configs.Logger.Info("Push notification would be sent",
		zap.Any("notification", notification))
}

// sendAdminNotification sends notifications to administrators
func (s *NotificationService) sendAdminNotification(notification *models.Notification) {
	// Admin email list (in a real implementation, get from config or DB)
	adminEmails := []string{"shipowner@wekeepgrowing.com"}

	for _, email := range adminEmails {
		// Send email
		err := EmailSvc.SendEmail(
			configs.Configs.Email.SenderEmail,
			email,
			fmt.Sprintf("[ADMIN ALERT] %s", notification.Title),
			fmt.Sprintf("<h1>%s</h1><p>%s</p><pre>%s</pre>",
				notification.Title, notification.Content, notification.Data),
		)

		if err != nil {
			configs.Logger.Error("Failed to send admin notification",
				zap.Error(err), zap.String("email", email))
		}
	}
}

// Global instance of NotificationService
var NotificationSvc = NewNotificationService()
