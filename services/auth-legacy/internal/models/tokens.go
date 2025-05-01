package models

import (
	"gorm.io/gorm"
	"time"
)

// Token represents an authentication token issued to a user
// Used for managing user authentication
type Token struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   uint      `gorm:"index;not null" json:"group_id"`             // Associated token group
	Token     string    `gorm:"size:250;not null" json:"token"`             // Encrypted token value
	TokenType string    `gorm:"size:50;default:'access'" json:"token_type"` // Type of token (access, refresh)
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`                 // Expiration timestamp

	// Standard metadata fields
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	TokenGroup *TokenGroup `gorm:"foreignKey:GroupID;references:ID;constraint:OnDelete:CASCADE" json:"token_group,omitempty"`
}
