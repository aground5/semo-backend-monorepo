package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// File represents a file record associated with an Item.
type File struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	FileName      string    `gorm:"type:varchar(255);not null" json:"file_name"`
	FileExtension string    `gorm:"type:varchar(50)" json:"file_extension"`
	ItemID        string    `gorm:"type:char(13);not null" json:"item_id"`
	ContentType   string    `gorm:"type:varchar(255)" json:"content_type"`

	Item *Item `gorm:"foreignKey:ItemID;references:ID" json:"item,omitempty"`

	// CreatedAt, UpdatedAt, DeletedAt는 GORM의 기본 메타 필드입니다.
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
