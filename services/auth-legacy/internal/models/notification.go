package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// NotificationType defines types of notifications that can be sent to users
// Used by NotificationService to categorize notifications
type NotificationType string

const (
	// Authentication notifications
	NotificationTypeNewLogin           NotificationType = "NEW_LOGIN"           // New login detected
	NotificationTypeUnknownDevice      NotificationType = "UNKNOWN_DEVICE"      // Login from unknown device
	NotificationTypeSuspiciousActivity NotificationType = "SUSPICIOUS_ACTIVITY" // Suspicious account activity
	NotificationTypeAccountLocked      NotificationType = "ACCOUNT_LOCKED"      // Account has been locked
	NotificationTypeLoginBlockedIP     NotificationType = "LOGIN_BLOCKED_IP"    // Login blocked due to IP

	// Security settings notifications
	NotificationTypeTwoFactorEnabled  NotificationType = "TWO_FACTOR_ENABLED"  // 2FA was enabled
	NotificationTypeTwoFactorDisabled NotificationType = "TWO_FACTOR_DISABLED" // 2FA was disabled
	NotificationTypePasswordChanged   NotificationType = "PASSWORD_CHANGED"    // Password was changed
	NotificationTypeNewDevice         NotificationType = "NEW_DEVICE"          // New device added to trusted devices
)

// NotificationChannel defines how notifications are delivered
// Used by NotificationService to determine delivery method
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "EMAIL"  // Send via email
	NotificationChannelSMS   NotificationChannel = "SMS"    // Send via SMS
	NotificationChannelPush  NotificationChannel = "PUSH"   // Send as push notification
	NotificationChannelInApp NotificationChannel = "IN_APP" // Display in application UI
	NotificationChannelAdmin NotificationChannel = "ADMIN"  // Send to system administrators
)

// Notification stores user notification information
// Used by NotificationService to manage and deliver notifications
type Notification struct {
	ID        uint                `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *string             `gorm:"type:char(12);index" json:"user_id,omitempty"` // Recipient (nil = admin notification)
	Type      NotificationType    `gorm:"size:50;not null" json:"type"`                 // Notification type
	Channel   NotificationChannel `gorm:"size:20;not null" json:"channel"`              // Delivery channel
	Title     string              `gorm:"size:200" json:"title"`                        // Notification title
	Content   string              `gorm:"type:text" json:"content"`                     // Notification content
	Data      datatypes.JSON      `gorm:"type:jsonb" json:"data"`                       // Additional structured data
	Read      bool                `gorm:"default:false" json:"read"`                    // Whether notification has been read
	SentAt    *time.Time          `json:"sent_at,omitempty"`                            // When notification was sent
	ReadAt    *time.Time          `json:"read_at,omitempty"`                            // When notification was read
	CreatedAt time.Time           `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time           `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt      `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// NotificationPreference stores user preferences for receiving notifications
// Used by NotificationService to respect user communication preferences
type NotificationPreference struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         string         `gorm:"type:char(12);uniqueIndex" json:"user_id"` // User ID
	EmailEnabled   bool           `gorm:"default:true" json:"email_enabled"`        // Allow email notifications
	SMSEnabled     bool           `gorm:"default:false" json:"sms_enabled"`         // Allow SMS notifications
	PushEnabled    bool           `gorm:"default:true" json:"push_enabled"`         // Allow push notifications
	InAppEnabled   bool           `gorm:"default:true" json:"in_app_enabled"`       // Allow in-app notifications
	SecurityAlerts bool           `gorm:"default:true" json:"security_alerts"`      // Receive security-related alerts
	LoginAlerts    bool           `gorm:"default:true" json:"login_alerts"`         // Receive login-related alerts
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}
