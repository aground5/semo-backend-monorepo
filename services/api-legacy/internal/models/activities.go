package models

import (
	"time"
)

type Activity struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProfileID   string `gorm:"type:char(12);" json:"profile_id"`
	DocumentID  string `gorm:"type:char(13);" json:"document_id"`
	Type        string `gorm:"type:varchar(100);" json:"type"`
	Description string `gorm:"type:varchar(250);" json:"description"`

	Profile  *Profile `gorm:"foreignKey:ProfileID;references:ID" json:"profile,omitempty"`
	Document *Item    `gorm:"foreignKey:DocumentID;references:ID" json:"document,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Activity) TableName() string {
	return "activities"
}
