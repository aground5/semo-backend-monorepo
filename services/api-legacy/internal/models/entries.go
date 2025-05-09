package models

import (
	"time"

	"gorm.io/gorm"
)

type Entry struct {
	ID         string `gorm:"type:char(13);primaryKey" json:"id"`
	Name       string `gorm:"type:varchar(255);not null" json:"name"`
	TaskID     string `gorm:"type:char(13);primaryKey" json:"task_id"`
	RootTaskID string `gorm:"type:char(13);primaryKey" json:"root_task_id"`

	CreatedBy string   `gorm:"type:char(12);" json:"created_by"`
	GrantedTo string   `gorm:"type:char(12);" json:"granted_to"`
	Scope     []string `gorm:"type:varchar(50)[]" json:"scope"`

	Creator *Profile `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	Grantee *Profile `gorm:"foreignKey:GrantedTo;references:ID" json:"grantee,omitempty"`

	Task     *Item `gorm:"foreignKey:TaskID;references:ID" json:"task,omitempty"`
	RootTask *Item `gorm:"foreignKey:RootTaskID;references:ID" json:"root_task,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
