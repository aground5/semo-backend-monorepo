package models

import (
	"gorm.io/gorm"
	"time"
)

type UserTests struct {
	ID       int    `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID   string `gorm:"type:char(13);not null" json:"task_id"`
	Question string `gorm:"type:text;not null" json:"question"`
	Answer   string `gorm:"type:text;not null" json:"answer"`
	UserData string `gorm:"type:text;not null" json:"user_data"`
	UserID   string `gorm:"type:char(12)" json:"user_id"`

	Task *Item `gorm:"foreignKey:TaskID;references:ID" json:"task,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (UserTests) TableName() string {
	return "public.user_tests"
}
