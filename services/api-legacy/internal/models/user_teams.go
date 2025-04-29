package models

import (
	"time"
)

type UserTeam struct {
	UserID   string    `gorm:"type:char(12);primaryKey" json:"user_id"`
	TeamID   string    `gorm:"type:char(12);primaryKey" json:"team_id"`
	Nickname string    `gorm:"type:varchar(250);" json:"nickname"`
	JoinedAt time.Time `gorm:"type:timestamp;" json:"joined_at"`

	Team    *Team    `gorm:"foreignKey:TeamID;references:ID;constraint:OnDelete:CASCADE;" json:"team"`
	Profile *Profile `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE;" json:"profile"`
}

func (UserTeam) TableName() string {
	return "public.user_teams"
}
