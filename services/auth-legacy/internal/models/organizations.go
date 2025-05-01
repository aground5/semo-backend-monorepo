package models

import (
	"gorm.io/gorm"
	"time"
)

// Organization represents an organizational unit within the system
// Used by InitialService for organization management
type Organization struct {
	ID           string `gorm:"type:char(12);primaryKey" json:"id"`          // Organization unique ID
	Name         string `gorm:"size:250;not null" json:"name"`               // Organization name
	Plan         string `gorm:"size:50;not null;default:'free'" json:"plan"` // Subscription plan (free, pro, enterprise)
	I18nLanguage string `gorm:"size:50;default:'ko'" json:"i18n_language"`   // Default language
	PhotoURL     string `gorm:"size:250" json:"photo_url"`                   // Organization logo/photo URL
	CreatedBy    string `gorm:"type:char(12)" json:"created_by"`             // User who created the organization
	Status       string `gorm:"size:50;default:'requested'" json:"status"`   // Organization status (requested, active, suspended)

	ProvisionedAt *time.Time `gorm:"type:timestamp;" json:"provisioned_at,omitempty"` // When organization was provisioned

	// Standard metadata fields
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Creator *User `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
}
