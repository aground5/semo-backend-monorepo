package logics

import (
	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"go.uber.org/zap"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

// CaptchaService provides CAPTCHA-related functionality
type CaptchaService struct{}

// NewCaptchaService creates a new CaptchaService instance
func NewCaptchaService() *CaptchaService {
	return &CaptchaService{}
}

// CaptchaResponse represents the result of CAPTCHA generation
type CaptchaResponse struct {
	ChallengeID string `json:"challenge_id"`
	ImageBase64 string `json:"image_base64,omitempty"` // For image CAPTCHAs
	Question    string `json:"question,omitempty"`     // For math problems or questions
	Type        string `json:"type"`                   // "image", "math", etc.
	ExpireIn    int    `json:"expire_in"`              // Expiration time in seconds
}

// VerifyResponse represents the result of CAPTCHA verification
type VerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// GenerateCaptcha creates a new CAPTCHA challenge
func (s *CaptchaService) GenerateCaptcha(ip, userAgent string) (*CaptchaResponse, error) {
	// 1. Check if too many challenges have been generated (rate limiting)
	recentChallenges, err := s.getRecentChallengeCount(ip)
	if err != nil {
		return nil, err
	}
	if recentChallenges > 10 {
		return nil, fmt.Errorf("too many CAPTCHA requests from this IP")
	}

	// 2. Generate challenge
	// Simple logic: 50% chance for image or math problem
	var challengeType string
	var answer string
	var imageBase64 string
	var question string

	randInt, _ := rand.Int(rand.Reader, big.NewInt(100))
	if randInt.Int64() < 50 {
		challengeType = "image"
		answer, imageBase64, err = s.generateImageCaptcha()
		if err != nil {
			return nil, err
		}
	} else {
		challengeType = "math"
		answer, question, err = s.generateMathCaptcha()
		if err != nil {
			return nil, err
		}
	}

	// 3. Generate challenge ID
	challengeID, err := s.generateChallengeID()
	if err != nil {
		return nil, err
	}

	// 4. Store hashed answer
	hashedAnswer := s.hashAnswer(answer)
	expiresAt := time.Now().Add(10 * time.Minute)

	challenge := &models.CaptchaChallenge{
		ChallengeID:   challengeID,
		ChallengeType: challengeType,
		Answer:        hashedAnswer,
		IP:            ip,
		UserAgent:     userAgent,
		Used:          false,
		AttemptCount:  0,
		ExpiresAt:     expiresAt,
	}

	if err := repositories.DBS.Postgres.Create(challenge).Error; err != nil {
		return nil, fmt.Errorf("failed to store CAPTCHA challenge: %w", err)
	}

	// 5. Return response
	response := &CaptchaResponse{
		ChallengeID: challengeID,
		ImageBase64: imageBase64,
		Question:    question,
		Type:        challengeType,
		ExpireIn:    600, // 10 minutes
	}

	return response, nil
}

// VerifyCaptcha verifies a submitted CAPTCHA response
func (s *CaptchaService) VerifyCaptcha(challengeID, response, ip, userAgent string) (*VerifyResponse, error) {
	// 1. Find the challenge
	var challenge models.CaptchaChallenge
	if err := repositories.DBS.Postgres.Where("challenge_id = ?", challengeID).First(&challenge).Error; err != nil {
		return &VerifyResponse{Success: false, Message: "Invalid challenge ID"}, nil
	}

	// 2. Check if expired
	if time.Now().After(challenge.ExpiresAt) {
		return &VerifyResponse{Success: false, Message: "Challenge has expired"}, nil
	}

	// 3. Check if already used
	if challenge.Used {
		return &VerifyResponse{Success: false, Message: "Challenge has already been used"}, nil
	}

	// 4. Increment attempt count
	repositories.DBS.Postgres.Model(&challenge).
		UpdateColumn("attempt_count", challenge.AttemptCount+1)

	if challenge.AttemptCount >= 5 {
		// Too many attempts - invalidate the challenge
		repositories.DBS.Postgres.Model(&challenge).
			UpdateColumn("used", true)
		return &VerifyResponse{Success: false, Message: "Too many attempts"}, nil
	}

	// 5. Verify the response
	hashedResponse := s.hashAnswer(response)
	if hashedResponse != challenge.Answer {
		// Record verification failure
		verification := &models.CaptchaVerification{
			ChallengeID: challengeID,
			Response:    response,
			Success:     false,
			IP:          ip,
			UserAgent:   userAgent,
		}
		repositories.DBS.Postgres.Create(verification)

		return &VerifyResponse{Success: false, Message: "Incorrect response"}, nil
	}

	// 6. Success - mark the challenge as used
	repositories.DBS.Postgres.Model(&challenge).
		UpdateColumn("used", true)

	// Record verification success
	verification := &models.CaptchaVerification{
		ChallengeID: challengeID,
		Response:    response,
		Success:     true,
		IP:          ip,
		UserAgent:   userAgent,
	}
	repositories.DBS.Postgres.Create(verification)

	return &VerifyResponse{Success: true, Message: "CAPTCHA verified successfully"}, nil
}

// IsCaptchaRequired determines if CAPTCHA is required for the given IP and email
func (s *CaptchaService) IsCaptchaRequired(ip, email string) bool {
	// 1. Check previous login failures
	var failCount int64
	repositories.DBS.Postgres.Model(&models.LoginAttempt{}).
		Where("ip = ? AND success = ? AND created_at > ?",
			ip, false, time.Now().Add(-24*time.Hour)).
		Count(&failCount)

	if failCount >= 3 {
		return true
	}

	// 2. Check failures for the email (if provided)
	if email != "" {
		var emailFailCount int64
		repositories.DBS.Postgres.Model(&models.LoginAttempt{}).
			Where("email = ? AND success = ? AND created_at > ?",
				email, false, time.Now().Add(-24*time.Hour)).
			Count(&emailFailCount)

		if emailFailCount >= 3 {
			return true
		}
	}

	// 3. Check for high-risk IP
	var blockedIPCount int64
	repositories.DBS.Postgres.Model(&models.BlockedIP{}).
		Where("ip = ?", ip).
		Or("ip LIKE ?", ip[:strings.LastIndex(ip, ".")+1]+"%"). // Same subnet
		Count(&blockedIPCount)

	if blockedIPCount > 0 {
		return true
	}

	// 4. Check for abnormally high recent login attempts
	var recentAttempts int64
	repositories.DBS.Postgres.Model(&models.LoginAttempt{}).
		Where("ip = ? AND created_at > ?",
			ip, time.Now().Add(-1*time.Hour)).
		Count(&recentAttempts)

	if recentAttempts > 10 {
		return true
	}

	return false
}

// Private helper methods

// generateImageCaptcha creates an image CAPTCHA with text
func (s *CaptchaService) generateImageCaptcha() (string, string, error) {
	// 1. Generate random text (6 characters)
	captchaText, err := s.generateRandomText(6)
	if err != nil {
		return "", "", err
	}

	// 2. Create image
	width, height := 200, 80
	dc := gg.NewContext(width, height)

	// Set background color
	dc.SetRGB(0.97, 0.97, 0.97)
	dc.Clear()

	// Add noise to background
	s.addNoise(dc, width, height)

	// Draw text
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return "", "", err
	}

	face := truetype.NewFace(font, &truetype.Options{
		Size: 36,
	})
	dc.SetFontFace(face)

	// Draw each character with different color and angle
	for i, char := range captchaText {
		// Random color
		r := 0.1 + 0.6*float64(i)/float64(len(captchaText))
		g := 0.1 + 0.5*float64(len(captchaText)-i)/float64(len(captchaText))
		b := 0.2 + 0.5*math.Sin(float64(i))
		dc.SetRGB(r, g, b)

		// Position and angle
		angle := -0.2 + 0.4*float64(i)/float64(len(captchaText))
		x := float64(width)/8 + float64(i)*float64(width)/8
		y := float64(height)/2 + 10*math.Sin(float64(i))

		// Rotate and draw text
		dc.RotateAbout(angle, x, y)
		dc.DrawStringAnchored(string(char), x, y, 0.5, 0.5)
		dc.RotateAbout(-angle, x, y)
	}

	// Add lines
	for i := 0; i < 4; i++ {
		dc.SetRGBA(0.5, 0.5, 0.5, 0.5)
		dc.SetLineWidth(1)

		// Use crypto/rand with Int() for secure random number generation
		randInt1, err := rand.Int(rand.Reader, big.NewInt(int64(height)))
		if err != nil {
			// Error handling
			configs.Logger.Fatal("Random number generation failed", zap.Error(err))
		}

		randInt2, err := rand.Int(rand.Reader, big.NewInt(int64(height)))
		if err != nil {
			// Error handling
			configs.Logger.Fatal("Random number generation failed", zap.Error(err))
		}

		y1 := float64(randInt1.Int64())
		y2 := float64(randInt2.Int64())

		dc.DrawLine(0, y1, float64(width), y2)
		dc.Stroke()
	}

	// Encode the image to PNG
	buf := new(bytes.Buffer)
	err = dc.EncodePNG(buf)
	if err != nil {
		return "", "", err
	}

	// Encode image to Base64
	base64Img := base64.StdEncoding.EncodeToString(buf.Bytes())

	return captchaText, base64Img, nil
}

// addNoise adds noise to the image
func (s *CaptchaService) addNoise(dc *gg.Context, width, height int) {
	for i := 0; i < 1000; i++ {
		// Generate x coordinate (0 ~ width-1)
		xRand, err := rand.Int(rand.Reader, big.NewInt(int64(width)))
		if err != nil {
			configs.Logger.Fatal("Random number generation failed", zap.Error(err))
		}
		x := float64(xRand.Int64())

		// Generate y coordinate (0 ~ height-1)
		yRand, err := rand.Int(rand.Reader, big.NewInt(int64(height)))
		if err != nil {
			configs.Logger.Fatal("Random number generation failed", zap.Error(err))
		}
		y := float64(yRand.Int64())

		// Generate RGB values (0 ~ 99)
		rRand, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			configs.Logger.Fatal("Random number generation failed", zap.Error(err))
		}
		r := float64(rRand.Int64()) / 100.0

		gRand, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			configs.Logger.Fatal("Random number generation failed", zap.Error(err))
		}
		g := float64(gRand.Int64()) / 100.0

		bRand, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			configs.Logger.Fatal("Random number generation failed", zap.Error(err))
		}
		b := float64(bRand.Int64()) / 100.0

		dc.SetRGBA(r, g, b, 0.3)
		dc.DrawPoint(x, y, 1)
		dc.Fill()
	}
}

// generateMathCaptcha creates a math problem CAPTCHA
func (s *CaptchaService) generateMathCaptcha() (string, string, error) {
	// 1. Generate a simple math problem
	var a, b int
	var op string
	var answer string
	var question string

	randInt, err := rand.Int(rand.Reader, big.NewInt(4))
	if err != nil {
		return "", "", err
	}

	// Generate two-digit numbers
	aInt, _ := rand.Int(rand.Reader, big.NewInt(90))
	a = int(aInt.Int64()) + 10
	bInt, _ := rand.Int(rand.Reader, big.NewInt(90))
	b = int(bInt.Int64()) + 10

	switch randInt.Int64() {
	case 0:
		// Addition
		op = "+"
		answer = strconv.Itoa(a + b)
		question = fmt.Sprintf("What is %d %s %d?", a, op, b)
	case 1:
		// Subtraction
		op = "-"
		// Ensure a is always larger than b
		if a < b {
			a, b = b, a
		}
		answer = strconv.Itoa(a - b)
		question = fmt.Sprintf("What is %d %s %d?", a, op, b)
	case 2:
		// Multiplication (simplified)
		op = "×"
		a = a%10 + 1 // 1-10 range
		b = b%10 + 1 // 1-10 range
		answer = strconv.Itoa(a * b)
		question = fmt.Sprintf("What is %d %s %d?", a, op, b)
	case 3:
		// Count characters
		text, err := s.generateRandomText(8)
		if err != nil {
			return "", "", err
		}

		// Generate random index (0 ~ len(text)-1)
		charIdxRand, err := rand.Int(rand.Reader, big.NewInt(int64(len(text))))
		if err != nil {
			return "", "", err
		}
		char := text[charIdxRand.Int64()]

		count := 0
		for _, c := range text {
			if c == rune(char) { // Fix: Convert char byte to rune for comparison
				count++
			}
		}
		answer = strconv.Itoa(count)
		question = fmt.Sprintf("How many times does the letter '%c' appear in '%s'?", char, text)
	}
	return answer, question, nil
}

// generateRandomText generates a random text of specified length
func (s *CaptchaService) generateRandomText(length int) (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Exclude easily confused characters (O, 0, I, 1)
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		randInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[randInt.Int64()]
	}
	return string(result), nil
}

// generateChallengeID generates a unique challenge ID
func (s *CaptchaService) generateChallengeID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashAnswer hashes a CAPTCHA response
func (s *CaptchaService) hashAnswer(answer string) string {
	// Normalize the answer (lowercase and trim spaces)
	normalizedAnswer := strings.ToLower(strings.TrimSpace(answer))
	hash := sha256.Sum256([]byte(normalizedAnswer))
	return hex.EncodeToString(hash[:])
}

// getRecentChallengeCount returns the count of recent challenges for an IP
func (s *CaptchaService) getRecentChallengeCount(ip string) (int64, error) {
	var count int64
	err := repositories.DBS.Postgres.Model(&models.CaptchaChallenge{}).
		Where("ip = ? AND created_at > ?", ip, time.Now().Add(-1*time.Hour)).
		Count(&count).Error
	return count, err
}

// Global instance of CaptchaService
var CaptchaSvc = NewCaptchaService()
