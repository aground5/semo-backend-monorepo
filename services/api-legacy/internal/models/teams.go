package models

import (
	"time"

	"gorm.io/gorm"
)

// Team represents a team entity in the database.
type Team struct {
	ID          string `gorm:"type:char(12);primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(250);" json:"name"`
	Description string `gorm:"type:varchar(250);" json:"description"`
	Status      string `gorm:"type:varchar(50);default:'active'" json:"status"`
	PhotoURL    string `gorm:"type:varchar(90);" json:"photo_url"`
	CreatedBy   string `gorm:"type:char(12);" json:"created_by"`
	ProjectID   string `gorm:"type:char(13);" json:"project_id"`

	// User relationships
	Creator *Profile   `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	Members []*Profile `gorm:"many2many:team_members;foreignKey:ID;joinForeignKey:TeamID;References:ID;joinReferences:UserID" json:"members,omitempty"`

	// Team membership details
	Memberships []*TeamMember `gorm:"foreignKey:TeamID;references:ID" json:"memberships,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Team) TableName() string {
	return "public.teams"
}

// TeamUpdate is used for partial updates on a team.
type TeamUpdate struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	PhotoURL    *string `json:"photo_url"`
}
