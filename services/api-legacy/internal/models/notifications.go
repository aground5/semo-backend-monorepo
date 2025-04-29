package models

import "time"

type Notification struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProfileID string `gorm:"type:char(12)" json:"profile_id"`
	Type      string `gorm:"type:varchar(250)" json:"type"`
	TaskID    string `gorm:"type:char(13)" json:"task_id"`
	Comment   string `gorm:"type:text" json:"comment"`

	Profile *Profile `gorm:"foreignKey:ProfileID;references:ID" json:"profile,omitempty"`
	Task    *Item    `gorm:"foreignKey:TaskID;references:ID" json:"task,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Notification) TableName() string {
	return "notifications"
}
