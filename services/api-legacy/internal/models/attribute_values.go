package models

import (
	"time"
)

type AttributeValue struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	AttributeID int    `gorm:"type:int;not null;uniqueIndex:unique_attribute_value_in_task" json:"attribute_id"`
	TaskID      string `gorm:"type:char(13);not null;uniqueIndex:unique_attribute_value_in_task" json:"task_id"`
	Value       string `gorm:"type:text" json:"value"`

	Attribute *Attribute `gorm:"foreignKey:AttributeID;references:ID" json:"attribute,omitempty"`
	Task      *Item      `gorm:"foreignKey:TaskID;references:ID" json:"task,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (AttributeValue) TableName() string {
	return "attribute_values"
}

// AttributeValueUpdate 구조체는 속성 값을 업데이트하는 데 사용됩니다.
type AttributeValueUpdate struct {
	AttributeID int    `json:"attribute_id" binding:"required"`
	TaskID      string `json:"task_id" binding:"required"`
	Value       string `json:"value"`
}
