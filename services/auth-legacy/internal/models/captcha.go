package models

import (
	"time"

	"gorm.io/gorm"
)

// CaptchaChallenge stores information about generated CAPTCHA challenges
// Used by CaptchaService to verify user responses
type CaptchaChallenge struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ChallengeID   string         `gorm:"size:64;uniqueIndex" json:"challenge_id"`       // Unique identifier for the challenge
	ChallengeType string         `gorm:"size:20;default:'image'" json:"challenge_type"` // Type: 'image', 'math', 'text'
	Answer        string         `gorm:"size:100" json:"answer"`                        // Hashed correct answer
	IP            string         `gorm:"size:50;index" json:"ip"`                       // IP that requested the CAPTCHA
	UserAgent     string         `gorm:"size:250" json:"user_agent"`                    // User agent that requested the CAPTCHA
	Used          bool           `gorm:"default:false" json:"used"`                     // Whether this challenge has been used
	AttemptCount  int            `gorm:"default:0" json:"attempt_count"`                // Number of verification attempts
	ExpiresAt     time.Time      `json:"expires_at"`                                    // When the challenge expires
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// CaptchaVerification records attempts to verify CAPTCHA responses
// Used by CaptchaService to track verification attempts
type CaptchaVerification struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ChallengeID string         `gorm:"size:64;index" json:"challenge_id"` // Reference to the challenge
	Response    string         `gorm:"size:100" json:"response"`          // User's response (not storing the actual value)
	Success     bool           `gorm:"default:false" json:"success"`      // Whether verification succeeded
	IP          string         `gorm:"size:50;index" json:"ip"`           // IP that submitted the verification
	UserAgent   string         `gorm:"size:250" json:"user_agent"`        // User agent that submitted the verification
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
