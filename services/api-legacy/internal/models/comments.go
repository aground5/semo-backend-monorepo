package models

import (
	"gorm.io/gorm"
	"time"
)

type Comment struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	ItemID    string `gorm:"type:char(13)" json:"item_id"`
	ProfileID string `gorm:"type:char(12)" json:"profile_id"`
	Comment   string `gorm:"type:text" json:"comment"`

	Writer *Profile `gorm:"foreignKey:ProfileID;references:ID" json:"writer,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Comment) TableName() string {
	return "comments"
}
