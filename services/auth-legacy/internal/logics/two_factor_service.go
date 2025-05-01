package logics

import (
	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

// TOTP related constants
const (
	totpIssuer      = "AuthnServer"
	backupCodeCount = 8
	backupCodeLen   = 10
)

// TwoFactorService provides two-factor authentication functionality
type TwoFactorService struct {
	encryptionKey []byte // Key used for secret encryption
}

// NewTwoFactorService creates a new TwoFactorService
func NewTwoFactorService() *TwoFactorService {
	// Use SHA-256 to create a consistent length key
	hasher := sha256.New()
	hasher.Write([]byte(configs.Configs.Secrets.SessionSecret)) // Reuse session secret
	key := hasher.Sum(nil)

	return &TwoFactorService{
		encryptionKey: key,
	}
}

// GenerateSecret generates a new TOTP secret
func (s *TwoFactorService) GenerateSecret(userID, email string) (*otp.Key, error) {
	// Generate secret key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: email,
	})
	if err != nil {
		return nil, err
	}

	// Generate backup codes
	backupCodes, err := s.generateBackupCodes()
	if err != nil {
		return nil, err
	}

	// Encrypt the secret and backup codes
	encryptedSecret, err := s.encrypt(key.Secret())
	if err != nil {
		return nil, err
	}

	encryptedBackupCodes, err := s.encrypt(strings.Join(backupCodes, ","))
	if err != nil {
		return nil, err
	}

	// Store in DB (disabled state)
	twoFactorSecret := &models.TwoFactorSecret{
		UserID:      userID,
		Secret:      encryptedSecret,
		BackupCodes: encryptedBackupCodes,
		Enabled:     false,
	}

	// Check for existing record
	var existingSecret models.TwoFactorSecret
	result := repositories.DBS.Postgres.Where("user_id = ?", userID).First(&existingSecret)

	// Save (or update)
	if result.Error == nil {
		// Update existing record
		existingSecret.Secret = encryptedSecret
		existingSecret.BackupCodes = encryptedBackupCodes
		existingSecret.Enabled = false // Disable when resetting with new secret
		if err := repositories.DBS.Postgres.Save(&existingSecret).Error; err != nil {
			return nil, err
		}
	} else {
		// Create new record
		if err := repositories.DBS.Postgres.Create(twoFactorSecret).Error; err != nil {
			return nil, err
		}
	}

	// Store temporarily in Redis for verification
	ctx := context.Background()
	redisKey := fmt.Sprintf("2fa:setup:%s", userID)
	backupCodesJSON, _ := json.Marshal(backupCodes)

	// Valid for 10 minutes
	err = repositories.DBS.Redis.Set(ctx, redisKey, backupCodesJSON, 10*time.Minute).Err()
	if err != nil {
		configs.Logger.Error("Failed to store 2FA setup data in Redis", zap.Error(err))
	}

	return key, nil
}

// VerifyAndEnable verifies a TOTP code and enables 2FA
func (s *TwoFactorService) VerifyAndEnable(userID, code string) (bool, []string, error) {
	// 1. Get secret from DB
	var secret models.TwoFactorSecret
	if err := repositories.DBS.Postgres.Where("user_id = ?", userID).First(&secret).Error; err != nil {
		return false, nil, err
	}

	// 2. Decrypt secret
	decryptedSecret, err := s.decrypt(secret.Secret)
	if err != nil {
		return false, nil, err
	}

	// 3. Verify code
	valid := totp.Validate(code, decryptedSecret)
	if !valid {
		return false, nil, nil
	}

	// 4. Enable 2FA
	secret.Enabled = true
	if err := repositories.DBS.Postgres.Save(&secret).Error; err != nil {
		return false, nil, err
	}

	// 5. Get backup codes from Redis
	ctx := context.Background()
	redisKey := fmt.Sprintf("2fa:setup:%s", userID)
	backupCodesJSON, err := repositories.DBS.Redis.Get(ctx, redisKey).Result()
	if err != nil {
		// If not found in Redis, decrypt from DB
		decryptedBackupCodes, err := s.decrypt(secret.BackupCodes)
		if err != nil {
			return true, nil, nil // Enabled but can't return backup codes
		}
		backupCodes := strings.Split(decryptedBackupCodes, ",")
		return true, backupCodes, nil
	}

	// 6. Parse backup codes
	var backupCodes []string
	if err := json.Unmarshal([]byte(backupCodesJSON), &backupCodes); err != nil {
		return true, nil, nil // Enabled but can't return backup codes
	}

	// 7. Delete backup codes from Redis
	repositories.DBS.Redis.Del(ctx, redisKey)

	// Record attempt
	attempt := &models.TwoFactorAttempt{
		UserID:    userID,
		IP:        "", // Should be extracted from context
		UserAgent: "", // Should be extracted from context
		Success:   true,
		Code:      "*****", // Don't store the actual code
		Type:      "totp",
	}
	repositories.DBS.Postgres.Create(attempt)

	// Create notification
	s.createNotification(userID, models.NotificationTypeTwoFactorEnabled,
		"2단계 인증이 활성화되었습니다",
		"계정에 2단계 인증이 성공적으로 활성화되었습니다.")

	return true, backupCodes, nil
}

// VerifyCode verifies a TOTP code during login
func (s *TwoFactorService) VerifyCode(userID, code, ip, userAgent string) (bool, error) {
	// 1. Get secret from DB
	var secret models.TwoFactorSecret
	if err := repositories.DBS.Postgres.Where("user_id = ? AND enabled = ?", userID, true).First(&secret).Error; err != nil {
		return false, err
	}

	// 2. Check if this is a backup code
	if len(code) == backupCodeLen {
		return s.verifyBackupCode(userID, code, ip, userAgent)
	}

	// 3. Verify TOTP code
	decryptedSecret, err := s.decrypt(secret.Secret)
	if err != nil {
		return false, err
	}

	// 4. Validate code
	valid := totp.Validate(code, decryptedSecret)

	// 5. Record attempt
	attempt := &models.TwoFactorAttempt{
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		Success:   valid,
		Code:      "*****", // Don't store the actual code
		Type:      "totp",
	}
	repositories.DBS.Postgres.Create(attempt)

	return valid, nil
}

// Disable2FA disables a user's 2FA
func (s *TwoFactorService) Disable2FA(userID string) error {
	result := repositories.DBS.Postgres.Model(&models.TwoFactorSecret{}).
		Where("user_id = ?", userID).
		Update("enabled", false)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		// Create notification
		s.createNotification(userID, models.NotificationTypeTwoFactorDisabled,
			"2단계 인증이 비활성화되었습니다",
			"계정의 2단계 인증이 비활성화되었습니다. 보안을 위해 다시 활성화하는 것을 권장합니다.")
	}

	return nil
}

// CreateChallenge creates a 2FA challenge
func (s *TwoFactorService) CreateChallenge(userID, ip, userAgent string) (string, error) {
	// 1. Check if user has 2FA enabled
	var secret models.TwoFactorSecret
	if err := repositories.DBS.Postgres.Where("user_id = ? AND enabled = ?", userID, true).First(&secret).Error; err != nil {
		return "", err
	}

	// 2. Generate unique challenge ID
	challengeID := fmt.Sprintf("%s-%d", userID, time.Now().UnixNano())
	challengeID = base64.RawURLEncoding.EncodeToString([]byte(challengeID))

	// 3. Store challenge
	challenge := &models.TwoFactorChallenge{
		ChallengeID: challengeID,
		UserID:      userID,
		IP:          ip,
		UserAgent:   userAgent,
		Completed:   false,
		ExpiresAt:   time.Now().Add(15 * time.Minute),
	}

	if err := repositories.DBS.Postgres.Create(challenge).Error; err != nil {
		return "", err
	}

	return challengeID, nil
}

// CompleteChallenge completes a 2FA challenge
func (s *TwoFactorService) CompleteChallenge(challengeID, code, ip, userAgent string) (bool, error) {
	// 1. Find the challenge
	var challenge models.TwoFactorChallenge
	if err := repositories.DBS.Postgres.Where("challenge_id = ? AND completed = ? AND expires_at > ?",
		challengeID, false, time.Now()).First(&challenge).Error; err != nil {
		return false, err
	}

	// 2. Verify the code
	valid, err := s.VerifyCode(challenge.UserID, code, ip, userAgent)
	if err != nil {
		return false, err
	}

	// 3. Mark challenge as completed if valid
	if valid {
		challenge.Completed = true
		if err := repositories.DBS.Postgres.Save(&challenge).Error; err != nil {
			return false, err
		}
	}

	return valid, nil
}

// Is2FAEnabled checks if a user has 2FA enabled
func (s *TwoFactorService) Is2FAEnabled(userID string) (bool, error) {
	var count int64
	err := repositories.DBS.Postgres.Model(&models.TwoFactorSecret{}).
		Where("user_id = ? AND enabled = ?", userID, true).
		Count(&count).Error

	return count > 0, err
}

// Private helper methods

// encrypt encrypts a string using AES-GCM
func (s *TwoFactorService) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	// GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 encode
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts an AES-GCM encrypted string
func (s *TwoFactorService) decrypt(encryptedText string) (string, error) {
	// Base64 decode
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	// GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Extract nonce
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// generateBackupCodes generates backup codes
func (s *TwoFactorService) generateBackupCodes() ([]string, error) {
	codes := make([]string, backupCodeCount)
	for i := 0; i < backupCodeCount; i++ {
		code, err := s.generateRandomString(backupCodeLen)
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

// generateRandomString generates a random string of specified length
func (s *TwoFactorService) generateRandomString(length int) (string, error) {
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		// Fix: Use big.NewInt with len(chars) instead of bytes.NewBuffer
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[n.Int64()]
	}
	return string(result), nil
}

// verifyBackupCode verifies a backup code
func (s *TwoFactorService) verifyBackupCode(userID, code, ip, userAgent string) (bool, error) {
	// 1. Get backup codes from DB
	var secret models.TwoFactorSecret
	if err := repositories.DBS.Postgres.Where("user_id = ? AND enabled = ?", userID, true).First(&secret).Error; err != nil {
		return false, err
	}

	// 2. Decrypt backup codes
	decryptedBackupCodes, err := s.decrypt(secret.BackupCodes)
	if err != nil {
		return false, err
	}

	// 3. Verify the code
	backupCodes := strings.Split(decryptedBackupCodes, ",")
	codeFound := false
	validCodes := []string{}

	for _, backupCode := range backupCodes {
		if backupCode == code {
			codeFound = true
		} else if backupCode != "" {
			validCodes = append(validCodes, backupCode)
		}
	}

	// 4. If code is valid, remove it from available codes
	if codeFound {
		// Save remaining codes
		newBackupCodesStr := strings.Join(validCodes, ",")
		encryptedBackupCodes, err := s.encrypt(newBackupCodesStr)
		if err != nil {
			return false, err
		}

		// Update DB
		if err := repositories.DBS.Postgres.Model(&secret).
			UpdateColumn("backup_codes", encryptedBackupCodes).Error; err != nil {
			return false, err
		}
	}

	// 5. Record attempt
	attempt := &models.TwoFactorAttempt{
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		Success:   codeFound,
		Code:      "*****", // Don't store the actual code
		Type:      "backup",
	}
	repositories.DBS.Postgres.Create(attempt)

	return codeFound, nil
}

// createNotification creates a notification
func (s *TwoFactorService) createNotification(userID string, notificationType models.NotificationType, title, content string) {
	notification := &models.Notification{
		UserID:  &userID,
		Type:    notificationType,
		Channel: models.NotificationChannelEmail, // Default to email
		Title:   title,
		Content: content,
		Read:    false,
	}

	if err := repositories.DBS.Postgres.Create(notification).Error; err != nil {
		configs.Logger.Error("Failed to create 2FA notification", zap.Error(err))
	}
}

// Global instance of TwoFactorService
var TwoFactorSvc = NewTwoFactorService()
