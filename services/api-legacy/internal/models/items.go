package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Item struct {
	ID          string          `gorm:"type:char(13);primaryKey" json:"id"`
	ParentID    *string         `gorm:"type:char(13);" json:"parent_id"`
	Type        string          `gorm:"type:varchar(100);" json:"type"`
	Name        string          `gorm:"type:varchar(250);" json:"name"` // 태스크 이름
	Contents    string          `gorm:"type:text;" json:"contents"`     // 태스크 내용
	Objective   string          `gorm:"type:text;" json:"objective"`    // 태스크 목표
	Deliverable string          `gorm:"type:text;" json:"deliverable"`  // 태스크 예상 결과물
	Role        string          `gorm:"type:varchar(250);" json:"role"` // 태스크 역할
	CreatedBy   string          `gorm:"type:char(12);" json:"created_by"`
	Color       string          `gorm:"type:varchar(6);default:'000000'" json:"color"`
	Position    decimal.Decimal `gorm:"type:decimal(20,10);default:'0'" json:"position"`

	// self-referencing 관계
	Parent   *Item  `gorm:"foreignKey:ParentID;references:ID" json:"parent,omitempty"`
	Children []Item `gorm:"foreignKey:ParentID;references:ID" json:"children,omitempty"`

	// Many-to-Many
	Dependencies []Item `gorm:"many2many:item_dependencies;joinForeignKey:ItemID;joinReferences:DependencyID" json:"dependencies"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Item) TableName() string {
	return "items"
}

// ItemUpdate is used for partial updates of an item.
type ItemUpdate struct {
	ParentID    *string `json:"parent_id"`
	Name        *string `json:"name"`
	Contents    *string `json:"contents"`
	LeftItemID  *string `json:"left_item_id"`
	Objective   *string `json:"objective"`
	Deliverable *string `json:"deliverable"`
	Role        *string `json:"role"`
}

type ItemInvite struct {
	Email string `json:"email"`
}

type ItemDependencies struct {
	ItemID       string `gorm:"type:char(13);primaryKey" json:"item_id"`
	DependencyID string `gorm:"type:char(13);primaryKey" json:"dependency_id"`
}
