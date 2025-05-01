package models

import (
	"time"

	"gorm.io/gorm"
)

// TwoFactorSecret stores a user's 2FA secret key and settings
// Used by TwoFactorService to manage 2FA authentication
type TwoFactorSecret struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      string         `gorm:"type:char(12);uniqueIndex" json:"user_id"` // Associated user
	Secret      string         `gorm:"size:500;not null" json:"secret"`          // Encrypted TOTP secret
	BackupCodes string         `gorm:"type:text" json:"backup_codes"`            // Encrypted backup recovery codes
	Enabled     bool           `gorm:"default:false" json:"enabled"`             // Whether 2FA is enabled
	Method      string         `gorm:"size:20;default:'totp'" json:"method"`     // 2FA method (totp, sms, email)
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TwoFactorAttempt records 2FA verification attempts
// Used by TwoFactorService to track and monitor verification attempts
type TwoFactorAttempt struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    string         `gorm:"type:char(12);index" json:"user_id"` // Associated user
	IP        string         `gorm:"size:50;index" json:"ip"`            // Source IP address
	UserAgent string         `gorm:"size:250" json:"user_agent"`         // User agent information
	Success   bool           `gorm:"default:false" json:"success"`       // Whether verification succeeded
	Code      string         `gorm:"size:10" json:"code"`                // Obfuscated code (for logging only)
	Type      string         `gorm:"size:20" json:"type"`                // Type of code: 'totp', 'backup', 'sms'
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TwoFactorChallenge represents a 2FA challenge during authentication
// Used by TwoFactorService to manage 2FA verification flow
type TwoFactorChallenge struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ChallengeID string         `gorm:"size:64;uniqueIndex" json:"challenge_id"` // Unique challenge identifier
	UserID      string         `gorm:"type:char(12);index" json:"user_id"`      // Associated user
	IP          string         `gorm:"size:50;index" json:"ip"`                 // Source IP address
	UserAgent   string         `gorm:"size:250" json:"user_agent"`              // User agent information
	SessionData string         `gorm:"type:text" json:"session_data"`           // Encrypted session data
	Completed   bool           `gorm:"default:false" json:"completed"`          // Whether challenge was completed
	ExpiresAt   time.Time      `json:"expires_at"`                              // When challenge expires
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}
