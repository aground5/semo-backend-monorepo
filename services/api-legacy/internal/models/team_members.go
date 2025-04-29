package models

import (
	"time"

	"gorm.io/gorm"
)

// TeamMember represents the many-to-many relationship between users and teams
// This allows users to belong to multiple teams and teams to have multiple users
type TeamMember struct {
	ID        string `gorm:"type:char(12);primaryKey" json:"id"`
	UserID    string `gorm:"type:char(12);not null;index" json:"user_id"`
	TeamID    string `gorm:"type:char(12);not null;index" json:"team_id"`
	Role      string `gorm:"type:varchar(50);default:'member'" json:"role"` // e.g., member, admin, owner
	Status    string `gorm:"type:varchar(50);default:'active'" json:"status"`
	InvitedBy string `gorm:"type:char(12);" json:"invited_by"`

	// Relationships
	Profile *Profile `gorm:"foreignKey:UserID;references:ID" json:"profile,omitempty"`
	Team    *Team    `gorm:"foreignKey:TeamID;references:ID" json:"team,omitempty"`
	Inviter *Profile `gorm:"foreignKey:InvitedBy;references:ID" json:"inviter,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (TeamMember) TableName() string {
	return "public.team_members"
}

// Note: Profile 모델에는 이미 Teams 관계가 설정되어 있습니다:
// Teams []*Team `gorm:"many2many:team_members;foreignKey:ID;joinForeignKey:UserID;References:ID;joinReferences:TeamID" json:"teams,omitempty"`

// Team 모델에는 Members 관계가 설정되어 있습니다:
// Members []*Profile `gorm:"many2many:team_members;foreignKey:ID;joinForeignKey:TeamID;References:ID;joinReferences:UserID" json:"members,omitempty"`
