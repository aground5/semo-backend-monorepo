package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Attribute struct {
	ID         int             `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string          `gorm:"type:varchar(250);" json:"name"`
	RootTaskID string          `gorm:"type:char(13);not null" json:"root_task_id"`
	Type       string          `gorm:"type:varchar(250);" json:"type"`
	Config     datatypes.JSON  `gorm:"type:jsonb" json:"config"`
	Position   decimal.Decimal `gorm:"type:decimal(20,10)" json:"position"`

	RootTask *Item `gorm:"foreignKey:RootTaskID;references:ID" json:"root_task,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Attribute) TableName() string {
	return "attributes"
}

// AttributeUpdate is used for partial updates of an attribute.
type AttributeUpdate struct {
	Name       *string         `json:"name"`
	Type       *string         `json:"type"`
	Config     *datatypes.JSON `json:"config"`
	LeftAttrID *int            `json:"left_attr_id"`
}
