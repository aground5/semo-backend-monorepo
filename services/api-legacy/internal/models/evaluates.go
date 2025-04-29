package models

import (
	"gorm.io/gorm"
	"time"
)

type Evaluate struct {
	ID      int    `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID  string `gorm:"type:char(13)" json:"task_id"`
	IsGo    bool   `gorm:"type:boolean" json:"is_go"`
	Comment string `gorm:"type:text" json:"comment"`

	Task *Item `gorm:"foreignKey:TaskID;references:ID" json:"task,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Evaluate) TableName() string {
	return "evaluates"
}
