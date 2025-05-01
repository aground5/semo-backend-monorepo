package models

import (
	"gorm.io/gorm"
	"time"
)

// TokenGroup represents a set of related tokens for a user
// Used for managing user authentication sessions
type TokenGroup struct {
	ID     uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID string `gorm:"type:char(12);not null;index" json:"user_id"` // Associated user
	Name   string `gorm:"size:100" json:"name,omitempty"`              // Optional name/description for the token group
	Device string `gorm:"size:250" json:"device,omitempty"`            // Device information where tokens are used

	// Standard metadata fields
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	User   *User   `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Tokens []Token `gorm:"foreignKey:GroupID" json:"tokens,omitempty"`
}

// BeforeDelete automatically deletes all tokens in the group when the group is deleted
func (tg *TokenGroup) BeforeDelete(tx *gorm.DB) (err error) {
	err = tx.Where("group_id = ?", tg.ID).
		Delete(&Token{}).Error
	return
}
