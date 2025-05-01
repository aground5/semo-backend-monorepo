package models

import (
	"gorm.io/gorm"
	"time"
)

// User represents a system user
// Core model used by all authentication and user-related services
type User struct {
	ID                string     `gorm:"type:char(12);primaryKey" json:"id"`             // User unique ID
	Username          string     `gorm:"size:100;not null" json:"username"`              // Username for login
	Name              string     `gorm:"size:100;not null;default:''" json:"name"`       // Display name
	Email             string     `gorm:"size:250;not null;uniqueIndex" json:"email"`     // Email address
	Password          string     `gorm:"size:250;not null" json:"password"`              // Hashed password
	Hash              string     `gorm:"size:250;not null" json:"hash"`                  // Salt value
	EmailVerified     bool       `gorm:"default:false" json:"email_verified"`            // Whether email is verified
	AccountStatus     string     `gorm:"size:50;default:'active'" json:"account_status"` // Status: active, suspended, locked
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`                        // Last successful login
	LastLoginIP       string     `gorm:"size:50" json:"last_login_ip,omitempty"`         // IP of last login
	FailedLoginCount  int        `gorm:"default:0" json:"failed_login_count"`            // Consecutive failed login attempts
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`                  // Last password change

	// Standard metadata fields
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships (defined with preloading options)
	TokenGroups            []TokenGroup            `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"token_groups,omitempty"`
	Activities             []Activity              `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"activities,omitempty"`
	TrustedDevices         []TrustedDevice         `gorm:"foreignKey:UserID" json:"trusted_devices,omitempty"`
	NotificationPreference *NotificationPreference `gorm:"foreignKey:UserID" json:"notification_preference,omitempty"`
	TwoFactorSecret        *TwoFactorSecret        `gorm:"foreignKey:UserID" json:"two_factor_secret,omitempty"`
}
