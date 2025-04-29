package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type Profile struct {
	ID          string         `gorm:"type:char(12);primaryKey" json:"id"`
	Email       string         `gorm:"size:250;not null;unique" json:"email"`
	Name        string         `gorm:"type:varchar(250);" json:"name"`
	DisplayName string         `gorm:"type:varchar(250);" json:"display_name"`
	Biography   string         `gorm:"type:text;" json:"biography"`
	Timezone    string         `gorm:"type:varchar(250);" json:"timezone"`
	Status      string         `gorm:"type:varchar(50);" json:"status"`
	PhotoURL    string         `gorm:"type:varchar(90);" json:"photo_url"`
	Config      datatypes.JSON `gorm:"type:jsonb" json:"config"`

	// Team relationships
	Teams []*Team `gorm:"many2many:team_members;foreignKey:ID;joinForeignKey:UserID;References:ID;joinReferences:TeamID" json:"teams,omitempty"`
	// Team invitations and memberships
	TeamMemberships []*TeamMember `gorm:"foreignKey:UserID;references:ID" json:"team_memberships,omitempty"`
	// Teams created by this user
	CreatedTeams []*Team `gorm:"foreignKey:CreatedBy;references:ID" json:"created_teams,omitempty"`

	// 공통 메타 필드
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// ProfileUpdate is used for partial updates of an profile.
type ProfileUpdate struct {
	Name        *string         `json:"name"`
	DisplayName *string         `json:"display_name"`
	Biography   *string         `json:"biography"`
	Timezone    *string         `json:"timezone"`
	Config      *datatypes.JSON `json:"config"`
}

func (Profile) TableName() string {
	return "profiles"
}
