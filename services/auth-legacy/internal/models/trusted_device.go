package models

import (
	"time"

	"gorm.io/gorm"
)

// TrustedDevice stores information about a user's trusted devices
// Used by TrustedDeviceService to manage trusted devices
type TrustedDevice struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     string         `gorm:"type:char(12);index" json:"user_id"` // Associated user
	DeviceUID  string         `gorm:"size:36;index" json:"device_uid"`    // Device unique identifier
	DeviceName string         `gorm:"size:100" json:"device_name"`        // User-defined device name
	DeviceType string         `gorm:"size:50" json:"device_type"`         // Device type (mobile, tablet, desktop)
	UserAgent  string         `gorm:"size:250" json:"user_agent"`         // User agent information
	LastIP     string         `gorm:"size:50" json:"last_ip"`             // Last IP address used
	LastUsed   time.Time      `json:"last_used"`                          // When device was last used
	ExpiresAt  time.Time      `json:"expires_at"`                         // When trust expires
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// UnknownDeviceAlert stores information about logins from unrecognized devices
// Used by TrustedDeviceService to manage device alerts
type UnknownDeviceAlert struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      string         `gorm:"type:char(12);index" json:"user_id"`         // Associated user
	DeviceUID   string         `gorm:"size:36;index" json:"device_uid"`            // Device unique identifier
	IP          string         `gorm:"size:50" json:"ip"`                          // IP address
	UserAgent   string         `gorm:"size:250" json:"user_agent"`                 // User agent information
	Location    string         `gorm:"size:100" json:"location"`                   // Approximate location based on IP
	RiskScore   int            `gorm:"default:0" json:"risk_score"`                // Calculated risk score (0-100)
	AlertSent   bool           `gorm:"default:false" json:"alert_sent"`            // Whether alert was sent to user
	ConfirmedBy string         `gorm:"size:20;default:'none'" json:"confirmed_by"` // Who confirmed: 'user', 'admin', 'auto', 'none'
	Action      string         `gorm:"size:20;default:'pending'" json:"action"`    // Action taken: 'blocked', 'allowed', 'pending'
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}
