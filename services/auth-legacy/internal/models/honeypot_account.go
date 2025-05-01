package models

import (
	"time"

	"gorm.io/gorm"
)

// HoneypotAccount stores fake account information used for bot detection
// Used by HoneypotService to track and identify malicious activities
type HoneypotAccount struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Email     string         `gorm:"size:250;uniqueIndex" json:"email"` // Fake email address
	Username  string         `gorm:"size:100;not null" json:"username"` // Fake username
	Name      string         `gorm:"size:100;not null" json:"name"`     // Fake display name
	Password  string         `gorm:"size:250;not null" json:"password"` // Hashed password
	Hash      string         `gorm:"size:250;not null" json:"hash"`     // Salt value
	IsActive  bool           `gorm:"default:true" json:"is_active"`     // Whether account is active
	Notes     string         `gorm:"size:500" json:"notes"`             // Administrative notes
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Activities []HoneypotActivity `gorm:"foreignKey:AccountID" json:"activities,omitempty"`
}

// HoneypotActivity records attempts to access honeypot accounts
// Used by HoneypotService to analyze and respond to suspicious activities
type HoneypotActivity struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID    uint           `gorm:"index" json:"account_id"`         // Associated honeypot account
	IP           string         `gorm:"size:50;index" json:"ip"`         // Source IP address
	UserAgent    string         `gorm:"size:250" json:"user_agent"`      // User agent information
	ActivityType string         `gorm:"size:50" json:"activity_type"`    // Type of activity: 'login_attempt', 'password_reset', etc.
	Severity     int            `gorm:"default:1" json:"severity"`       // Severity level: 1 (low) to 5 (high)
	Details      string         `gorm:"type:text" json:"details"`        // JSON-formatted details about the activity
	BlockedIP    bool           `gorm:"default:false" json:"blocked_ip"` // Whether the IP was blocked as a result
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Account *HoneypotAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}
