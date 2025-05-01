package logics

import (
	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HoneypotService provides honeypot account functionality
type HoneypotService struct{}

// NewHoneypotService creates a new HoneypotService instance
func NewHoneypotService() *HoneypotService {
	return &HoneypotService{}
}

// CreateHoneypotAccount creates a new honeypot account
func (s *HoneypotService) CreateHoneypotAccount(email, username, name, password, notes string) (*models.HoneypotAccount, error) {
	// 1. Check if account already exists
	var existingAccount models.HoneypotAccount
	result := repositories.DBS.Postgres.Where("email = ?", email).First(&existingAccount)
	if result.Error == nil {
		return nil, fmt.Errorf("account with email %s already exists", email)
	} else if result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}

	// 2. Generate hash
	hash := make([]byte, 16)
	_, err := rand.Read(hash)
	if err != nil {
		return nil, err
	}
	hashStr := base64.StdEncoding.EncodeToString(hash)

	// 3. Hash the password + salt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password+hashStr), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 4. Create account
	honeypotAccount := &models.HoneypotAccount{
		Email:    email,
		Username: username,
		Name:     name,
		Password: string(hashedPassword),
		Hash:     hashStr,
		IsActive: true,
		Notes:    notes,
	}

	if err := repositories.DBS.Postgres.Create(honeypotAccount).Error; err != nil {
		return nil, err
	}

	configs.Logger.Info("Honeypot account created",
		zap.String("email", email),
		zap.String("username", username))

	return honeypotAccount, nil
}

// GetHoneypotAccounts returns all honeypot accounts
func (s *HoneypotService) GetHoneypotAccounts() ([]models.HoneypotAccount, error) {
	var accounts []models.HoneypotAccount
	err := repositories.DBS.Postgres.Find(&accounts).Error
	return accounts, err
}

// HandleHoneypotActivity records and processes honeypot account activity
func (s *HoneypotService) HandleHoneypotActivity(accountID uint, ip, userAgent, activityType string, details map[string]interface{}) error {
	// 1. Check if account exists
	var account models.HoneypotAccount
	if err := repositories.DBS.Postgres.First(&account, accountID).Error; err != nil {
		return err
	}

	// 2. Determine severity
	severity := s.determineSeverity(activityType, ip)

	// 3. Convert details to JSON
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	// 4. Record activity
	activity := &models.HoneypotActivity{
		AccountID:    accountID,
		IP:           ip,
		UserAgent:    userAgent,
		ActivityType: activityType,
		Severity:     severity,
		Details:      string(detailsJSON),
	}

	if err := repositories.DBS.Postgres.Create(activity).Error; err != nil {
		return err
	}

	// 5. Block IP for high severity
	if severity >= 4 {
		activity.BlockedIP = true
		if err := repositories.DBS.Postgres.Save(activity).Error; err != nil {
			configs.Logger.Error("Failed to update activity block status", zap.Error(err))
		}

		// Block the IP
		blockedIP := &models.BlockedIP{
			IP:           ip,
			Reason:       fmt.Sprintf("Honeypot activity: %s", activityType),
			BlockedUntil: time.Now().Add(24 * time.Hour),
			Permanent:    false,
		}

		if err := repositories.DBS.Postgres.Create(blockedIP).Error; err != nil {
			configs.Logger.Error("Failed to block IP", zap.Error(err))
		} else {
			configs.Logger.Warn("IP blocked due to honeypot activity",
				zap.String("ip", ip),
				zap.String("activity", activityType))
		}

		// Notify admin
		s.notifyAdminHoneypotActivity(activity, &account)
	}

	return nil
}

// IsHoneypotAccount checks if an email belongs to a honeypot account
func (s *HoneypotService) IsHoneypotAccount(email string) (bool, uint) {
	var account models.HoneypotAccount
	result := repositories.DBS.Postgres.Where("email = ? AND is_active = ?", email, true).First(&account)
	if result.Error != nil {
		return false, 0
	}
	return true, account.ID
}

// ValidateHoneypotCredentials validates honeypot account credentials
func (s *HoneypotService) ValidateHoneypotCredentials(email, password string) (bool, uint) {
	var account models.HoneypotAccount
	result := repositories.DBS.Postgres.Where("email = ? AND is_active = ?", email, true).First(&account)
	if result.Error != nil {
		return false, 0
	}

	// Credentials validation
	// Always fail authentication - providing credentials is considered bot activity
	return false, account.ID
}

// Private helper methods

// determineSeverity determines activity severity based on type and IP
func (s *HoneypotService) determineSeverity(activityType string, ip string) int {
	// Base severity
	severity := 1

	// Adjust based on activity type
	switch activityType {
	case "login_attempt":
		severity = 3
	case "password_reset":
		severity = 4
	case "api_access":
		severity = 5
	}

	// Check if IP is already blocked
	var blockedIP models.BlockedIP
	result := repositories.DBS.Postgres.Where("ip = ?", ip).First(&blockedIP)
	if result.Error == nil {
		// Already blocked IP - increase severity
		severity += 1
	}

	// Check recent honeypot activity from this IP
	var recentCount int64
	repositories.DBS.Postgres.Model(&models.HoneypotActivity{}).
		Where("ip = ? AND created_at > ?", ip, time.Now().Add(-24*time.Hour)).
		Count(&recentCount)

	if recentCount > 0 {
		// Repeated activity - increase severity
		severity += int(recentCount)
	}

	// Limit severity range (1-5)
	if severity < 1 {
		severity = 1
	} else if severity > 5 {
		severity = 5
	}

	return severity
}

// notifyAdminHoneypotActivity notifies admins about honeypot activity
func (s *HoneypotService) notifyAdminHoneypotActivity(activity *models.HoneypotActivity, account *models.HoneypotAccount) {
	data := map[string]interface{}{
		"account_id":    activity.AccountID,
		"account_email": account.Email,
		"ip":            activity.IP,
		"activity_type": activity.ActivityType,
		"severity":      activity.Severity,
		"details":       activity.Details,
		"timestamp":     activity.CreatedAt,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		configs.Logger.Error("Failed to marshal notification data", zap.Error(err))
		return
	}

	notification := &models.Notification{
		UserID:  nil, // Admin notification
		Type:    models.NotificationTypeSuspiciousActivity,
		Channel: models.NotificationChannelAdmin,
		Title:   "Honeypot Activity Detected",
		Content: fmt.Sprintf("Honeypot account %s was accessed from IP %s", account.Email, activity.IP),
		Data:    dataJSON,
		Read:    false,
	}

	if err := repositories.DBS.Postgres.Create(notification).Error; err != nil {
		configs.Logger.Error("Failed to create notification", zap.Error(err))
	}
}

// Global instance of HoneypotService
var HoneypotSvc = NewHoneypotService()
