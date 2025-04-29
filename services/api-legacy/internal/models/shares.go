package models

import (
	"time"
)

type Share struct {
	ID         string `gorm:"type:uuid;primaryKey" json:"id"`
	RootTaskID string `gorm:"type:char(13);primaryKey" json:"root_task_id"`

	CreatedBy string `gorm:"type:char(12);" json:"created_by"`
	GrantedTo string `gorm:"type:char(12);" json:"granted_to"`

	Creator *Profile `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	Grantee *Profile `gorm:"foreignKey:GrantedTo;references:ID" json:"grantee,omitempty"`

	RootTask *Item `gorm:"foreignKey:RootTaskID;references:ID" json:"root_task,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}