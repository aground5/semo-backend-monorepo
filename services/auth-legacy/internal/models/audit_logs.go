package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

// AuditLog stores system audit events for security tracking and compliance
// Used by AuditLogService to record various system events
type AuditLog struct {
	ID      uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID  *string        `gorm:"type:char(12);index" json:"user_id,omitempty"` // Optional: some logs don't belong to users
	Type    AuditLogType   `gorm:"size:50;not null" json:"type"`                 // Type of audit event
	Content datatypes.JSON `gorm:"type:jsonb" json:"content"`                    // Structured event details

	// Standard metadata fields
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
}
