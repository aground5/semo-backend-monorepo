package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// Activity tracks user login sessions
// Used by RemoteAuthService to manage active sessions
type Activity struct {
	SessionID    string     `gorm:"type:char(52);primaryKey" json:"session_id"` // Unique session identifier
	UserID       string     `gorm:"type:char(12);primaryKey" json:"user_id"`    // Associated user
	TokenGroupID uint       `gorm:"index" json:"token_group_id,omitempty"`      // Associated token group
	IP           string     `gorm:"size:250" json:"ip"`                         // Source IP address
	UserAgent    string     `gorm:"size:250" json:"useragent"`                  // User agent information
	DeviceUID    *uuid.UUID `gorm:"type:char(36);" json:"device_uid"`           // Device unique identifier
	LoginAt      time.Time  `json:"login_at"`                                   // When session started
	LogoutAt     *time.Time `json:"logout_at,omitempty"`                        // When session ended (nil = active)
	LocationInfo string     `gorm:"size:250" json:"location_info,omitempty"`    // Location information
	DeviceInfo   string     `gorm:"size:250" json:"device_info,omitempty"`      // Device information

	// Standard metadata fields
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User       *User       `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
	TokenGroup *TokenGroup `gorm:"foreignKey:TokenGroupID;references:ID;constraint:OnDelete:SET NULL" json:"token_group,omitempty"`
}

func (Activity) TableName() string {
	return "login_activities"
}
