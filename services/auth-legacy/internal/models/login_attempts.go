package models

import (
	"time"

	"gorm.io/gorm"
)

// LoginAttempt tracks login attempts for security monitoring and bot prevention
// Used by BotPreventionService to detect suspicious login patterns
type LoginAttempt struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Email     string         `gorm:"size:250;index" json:"email"`                  // Email used in login attempt
	IP        string         `gorm:"size:50;index" json:"ip"`                      // Source IP address
	UserAgent string         `gorm:"size:250" json:"user_agent"`                   // User agent information
	Success   bool           `gorm:"index" json:"success"`                         // Whether login succeeded
	DeviceUID string         `gorm:"size:36;index" json:"device_uid"`              // Device unique identifier
	UserID    *string        `gorm:"type:char(12);index" json:"user_id,omitempty"` // User ID (only for successful logins)
	Location  string         `gorm:"size:100" json:"location"`                     // Approximate location (based on IP)
	RiskScore int            `gorm:"default:0" json:"risk_score"`                  // Calculated risk score (0-100)
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BlockedIP represents an IP address that is blocked from accessing the system
// Used by BotPreventionService for IP blocking functionality
type BlockedIP struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	IP           string         `gorm:"size:50;uniqueIndex" json:"ip"`  // Blocked IP address
	Reason       string         `gorm:"size:250" json:"reason"`         // Reason for blocking
	BlockedUntil time.Time      `gorm:"index" json:"blocked_until"`     // When the block expires
	Permanent    bool           `gorm:"default:false" json:"permanent"` // Whether block is permanent
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// DeviceFingerprint stores device information for identification and risk assessment
// Used by BotPreventionService to identify and track devices
type DeviceFingerprint struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceUID  string         `gorm:"size:36;uniqueIndex" json:"device_uid"` // Device unique identifier
	UserID     string         `gorm:"type:char(12);index" json:"user_id"`    // Associated user
	UserAgent  string         `gorm:"size:250" json:"user_agent"`            // User agent information
	IP         string         `gorm:"size:50" json:"ip"`                     // Last seen IP address
	Attributes string         `gorm:"type:text" json:"attributes"`           // JSON-formatted device attributes
	Trusted    bool           `gorm:"default:false" json:"trusted"`          // Whether device is trusted
	LastSeen   time.Time      `json:"last_seen"`                             // When device was last seen
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}
