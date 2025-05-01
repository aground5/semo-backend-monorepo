package logics

import (
	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BotPreventionService provides methods for bot detection and prevention
type BotPreventionService struct{}

// NewBotPreventionService creates a new BotPreventionService
func NewBotPreventionService() *BotPreventionService {
	return &BotPreventionService{}
}

// LoginRequest holds information about login request for analysis
type LoginRequest struct {
	Email     string
	DeviceUID *uuid.UUID
	IP        string
	UserAgent string
	SessionID string
}

// CheckLoginAllowed verifies if login should be allowed based on various security measures
func (s *BotPreventionService) CheckLoginAllowed(c echo.Context, req LoginRequest) (bool, string) {
	// 1. Check if IP is blocked
	blocked, reason := s.checkIPBlocked(req.IP)
	if blocked {
		configs.Logger.Warn("Login blocked - IP is in block list",
			zap.String("ip", req.IP),
			zap.String("reason", reason),
			zap.String("email", req.Email))
		return false, fmt.Sprintf("Login blocked: %s", reason)
	}

	// 2. Check for rate limiting
	if s.isRateLimited(req.Email, req.IP) {
		configs.Logger.Warn("Login blocked - rate limited",
			zap.String("ip", req.IP),
			zap.String("email", req.Email))
		return false, "Too many login attempts. Please try again later."
	}

	// 3. Get risk score of this login attempt
	riskScore := s.calculateRiskScore(req)
	if riskScore >= 80 {
		configs.Logger.Warn("Login blocked - high risk score",
			zap.String("ip", req.IP),
			zap.String("email", req.Email),
			zap.Int("risk_score", riskScore))
		return false, "Suspicious login activity detected. Please try again later or contact support."
	}

	return true, ""
}

// RecordLoginAttempt logs the login attempt for analytics and security monitoring
func (s *BotPreventionService) RecordLoginAttempt(req LoginRequest, success bool, userID *string) {
	// Create login attempt record
	deviceUIDString := ""
	if req.DeviceUID != nil {
		deviceUIDString = req.DeviceUID.String()
	}

	loginAttempt := &models.LoginAttempt{
		Email:     req.Email,
		IP:        req.IP,
		UserAgent: req.UserAgent,
		Success:   success,
		DeviceUID: deviceUIDString,
		UserID:    userID,
		Location:  s.getGeoLocation(req.IP),
		RiskScore: s.calculateRiskScore(req),
	}

	err := repositories.DBS.Postgres.Create(loginAttempt).Error
	if err != nil {
		configs.Logger.Error("Failed to record login attempt",
			zap.Error(err),
			zap.String("email", req.Email),
			zap.String("ip", req.IP))
	}

	// If failed attempt, increment failure count in Redis for rate limiting
	if !success {
		s.incrementFailedLoginCounter(req.Email, req.IP)

		// Check if we need to block the IP due to too many failures
		s.checkAndBlockIPIfNeeded(req.Email, req.IP)
	} else if req.DeviceUID != nil {
		// If successful login, update or create device fingerprint
		s.updateDeviceFingerprint(req, userID)
	}
}

// CalculateDeviceFingerprint generates a hash for a device based on provided attributes
func (s *BotPreventionService) CalculateDeviceFingerprint(c echo.Context) string {
	// Get various device attributes
	userAgent := c.Request().UserAgent()
	ip := c.RealIP()
	acceptHeader := c.Request().Header.Get("Accept")
	acceptLanguage := c.Request().Header.Get("Accept-Language")
	acceptEncoding := c.Request().Header.Get("Accept-Encoding")

	// Combine attributes to create a fingerprint
	fingerprintStr := fmt.Sprintf("%s|%s|%s|%s|%s",
		userAgent, ip, acceptHeader, acceptLanguage, acceptEncoding)

	// Create hash of the fingerprint
	hasher := sha256.New()
	hasher.Write([]byte(fingerprintStr))

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// Private helper methods

func (s *BotPreventionService) updateDeviceFingerprint(req LoginRequest, userID *string) {
	if req.DeviceUID == nil || userID == nil {
		return
	}

	deviceUID := req.DeviceUID.String()
	attributes := map[string]string{
		"user_agent": req.UserAgent,
		"ip":         req.IP,
	}

	attributesJSON, _ := json.Marshal(attributes)

	var fingerprint models.DeviceFingerprint
	result := repositories.DBS.Postgres.Where("device_uid = ?", deviceUID).First(&fingerprint)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create new fingerprint
			fingerprint = models.DeviceFingerprint{
				DeviceUID:  deviceUID,
				UserID:     *userID,
				UserAgent:  req.UserAgent,
				IP:         req.IP,
				Attributes: string(attributesJSON),
				LastSeen:   time.Now(),
				Trusted:    false, // New devices are not trusted by default
			}
			repositories.DBS.Postgres.Create(&fingerprint)
		}
	} else {
		// Update existing fingerprint
		repositories.DBS.Postgres.Model(&fingerprint).Updates(map[string]interface{}{
			"user_agent": req.UserAgent,
			"ip":         req.IP,
			"attributes": string(attributesJSON),
			"last_seen":  time.Now(),
		})
	}
}

func (s *BotPreventionService) checkIPBlocked(ip string) (bool, string) {
	var blockedIP models.BlockedIP
	result := repositories.DBS.Postgres.Where("ip = ?", ip).First(&blockedIP)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, ""
		}
		configs.Logger.Error("Error checking IP block status", zap.Error(result.Error))
		return false, ""
	}

	// Check if the block has expired
	if !blockedIP.Permanent && time.Now().After(blockedIP.BlockedUntil) {
		// Block has expired, remove it
		repositories.DBS.Postgres.Delete(&blockedIP)
		return false, ""
	}

	return true, blockedIP.Reason
}

func (s *BotPreventionService) isRateLimited(email, ip string) bool {
	ctx := context.Background()
	redisKey := fmt.Sprintf("rate_limit:login:%s:%s", email, ip)

	// Get current count from Redis
	count, err := repositories.DBS.Redis.Get(ctx, redisKey).Int()
	if err != nil {
		// Key doesn't exist or other error, so not rate limited
		return false
	}

	// Rate limit threshold (e.g., 5 attempts in 15 minutes)
	return count >= 5
}

func (s *BotPreventionService) calculateRiskScore(req LoginRequest) int {
	score := 0

	// 1. Check if email is frequently targeted
	emailAttempts := s.getRecentLoginAttempts("email", req.Email, 24*time.Hour)
	if emailAttempts > 10 {
		score += 20 // High number of attempts for this email
	}

	// 2. Check IP reputation
	ipAttempts := s.getRecentLoginAttempts("ip", req.IP, 24*time.Hour)
	if ipAttempts > 30 {
		score += 30 // This IP has made many attempts
	}

	// 3. Check geographic anomalies
	location := s.getGeoLocation(req.IP)
	if s.isUncommonLocation(req.Email, location) {
		score += 25 // Uncommon location for this user
	}

	// 4. Check for data center or proxy IP
	if s.isDataCenterIP(req.IP) {
		score += 20 // Data center IPs are more suspicious
	}

	// 5. Check for unusual login patterns
	if s.isUnusualLoginTime(req.Email) {
		score += 10 // Unusual time of day for this user
	}

	// 6. Check user agent for bot signatures
	if s.hasAnomalousUserAgent(req.UserAgent) {
		score += 25 // User agent looks like a bot
	}

	// Cap the score at 100
	if score > 100 {
		score = 100
	}

	return score
}

// Helper functions to support risk scoring

func (s *BotPreventionService) getRecentLoginAttempts(field, value string, duration time.Duration) int {
	var count int64
	since := time.Now().Add(-duration)

	repositories.DBS.Postgres.Model(&models.LoginAttempt{}).
		Where(fmt.Sprintf("%s = ? AND created_at > ?", field), value, since).
		Count(&count)

	return int(count)
}

func (s *BotPreventionService) getGeoLocation(ip string) string {
	// In a real implementation, this would use a GeoIP database like MaxMind
	// For now, just extract netblock prefix as a simple location indicator
	ipObj := net.ParseIP(ip)
	if ipObj == nil {
		return "unknown"
	}

	if ipObj.IsLoopback() || ipObj.IsPrivate() {
		return "local"
	}

	// Simple prefix extraction
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return fmt.Sprintf("%s.%s.0.0", parts[0], parts[1])
	}

	return "unknown"
}

func (s *BotPreventionService) isUncommonLocation(email, location string) bool {
	// Get user's common locations
	var commonLocations []string
	var attempts []models.LoginAttempt

	// Find successful logins for this email in the last 30 days
	repositories.DBS.Postgres.
		Where("email = ? AND success = ? AND created_at > ?",
			email, true, time.Now().AddDate(0, 0, -30)).
		Find(&attempts)

	// Count occurrences of each location
	locationCounts := make(map[string]int)
	for _, attempt := range attempts {
		locationCounts[attempt.Location]++
	}

	// Locations with more than 2 logins are considered "common"
	for loc, count := range locationCounts {
		if count >= 2 {
			commonLocations = append(commonLocations, loc)
		}
	}

	// If we have common locations and the current location is not among them
	if len(commonLocations) > 0 {
		for _, commonLoc := range commonLocations {
			if commonLoc == location {
				return false
			}
		}
		return true
	}

	// If no common locations established yet, don't consider it uncommon
	return false
}

func (s *BotPreventionService) isDataCenterIP(ip string) bool {
	// In a real implementation, this would check against known data center IP ranges
	// For demonstration, we'll use a simplified check

	// Check for common cloud provider ranges (very simplified example)
	cloudPrefixes := []string{
		"34.0.", "35.0.", // Example Google Cloud prefixes
		"52.0.", "54.0.", // Example AWS prefixes
		"40.0.", "20.0.", // Example Azure prefixes
	}

	for _, prefix := range cloudPrefixes {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}

	return false
}

func (s *BotPreventionService) isUnusualLoginTime(email string) bool {
	// Get current hour in 24-hour format (0-23)
	currentHour := time.Now().Hour()

	// Define unusual hours (e.g., very late night/early morning)
	unusualHours := []int{1, 2, 3, 4, 5} // 1 AM to 5 AM

	for _, hour := range unusualHours {
		if currentHour == hour {
			// Check if user normally logs in during this hour
			var count int64
			repositories.DBS.Postgres.Model(&models.LoginAttempt{}).
				Where("email = ? AND success = ? AND EXTRACT(HOUR FROM created_at) = ?",
					email, true, currentHour).
				Count(&count)

			// If user has logged in successfully during this hour before, it's not unusual
			return count == 0
		}
	}

	return false
}

func (s *BotPreventionService) hasAnomalousUserAgent(userAgent string) bool {
	userAgentLower := strings.ToLower(userAgent)

	// Check for bot signatures in user agent
	botSignatures := []string{
		"bot", "crawler", "spider", "http", "curl", "wget",
		"python", "requests", "java", "phantomjs", "headless",
	}

	for _, sig := range botSignatures {
		if strings.Contains(userAgentLower, sig) {
			return true
		}
	}

	// Check for abnormal user agent length
	if len(userAgent) < 20 || len(userAgent) > 500 {
		return true
	}

	return false
}

func (s *BotPreventionService) incrementFailedLoginCounter(email, ip string) {
	ctx := context.Background()
	redisKey := fmt.Sprintf("rate_limit:login:%s:%s", email, ip)

	// Increment counter with 15-minute expiry
	repositories.DBS.Redis.Incr(ctx, redisKey)
	repositories.DBS.Redis.Expire(ctx, redisKey, 15*time.Minute)

	// Also increment a longer-term counter for analytics
	longTermKey := fmt.Sprintf("fails:%s:%s", email, ip)
	repositories.DBS.Redis.Incr(ctx, longTermKey)
	repositories.DBS.Redis.Expire(ctx, longTermKey, 24*time.Hour)
}

func (s *BotPreventionService) checkAndBlockIPIfNeeded(email, ip string) {
	ctx := context.Background()
	longTermKey := fmt.Sprintf("fails:%s:%s", email, ip)

	// Get current count from Redis
	count, err := repositories.DBS.Redis.Get(ctx, longTermKey).Int()
	if err != nil {
		return
	}

	// If more than 10 failed attempts in 24 hours, block IP temporarily
	if count >= 10 {
		blockDuration := 1 * time.Hour
		reason := "Too many failed login attempts"

		// Check if this IP has been blocked before
		var previousBlocks int64
		repositories.DBS.Postgres.Model(&models.BlockedIP{}).
			Where("ip = ?", ip).
			Count(&previousBlocks)

		// Increase block duration for repeat offenders
		if previousBlocks > 0 {
			blockDuration = time.Duration(previousBlocks+1) * blockDuration
			if blockDuration > 24*time.Hour {
				blockDuration = 24 * time.Hour
			}
		}

		// Create block record
		blockedIP := models.BlockedIP{
			IP:           ip,
			Reason:       reason,
			BlockedUntil: time.Now().Add(blockDuration),
			Permanent:    false,
		}

		repositories.DBS.Postgres.Create(&blockedIP)

		configs.Logger.Warn("IP blocked due to failed login attempts",
			zap.String("ip", ip),
			zap.String("email", email),
			zap.Duration("duration", blockDuration))
	}
}

// Global instance of BotPreventionService
var BotPreventionSvc = NewBotPreventionService()
