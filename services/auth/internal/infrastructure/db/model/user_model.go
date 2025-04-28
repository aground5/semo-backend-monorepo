package model

import (
	"time"

	"gorm.io/gorm"
)

// UserModel 데이터베이스 ORM 모델
type UserModel struct {
	ID                string     `gorm:"type:char(12);primaryKey" json:"id"`
	Username          string     `gorm:"size:100;not null" json:"username"`
	Name              string     `gorm:"size:100;not null;default:''" json:"name"`
	Email             string     `gorm:"size:250;not null;uniqueIndex" json:"email"`
	Password          string     `gorm:"size:250;not null" json:"password"`
	Salt              string     `gorm:"size:250;not null" json:"salt"`
	EmailVerified     bool       `gorm:"default:false" json:"email_verified"`
	AccountStatus     string     `gorm:"size:50;default:'active'" json:"account_status"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP       string     `gorm:"size:50" json:"last_login_ip,omitempty"`
	FailedLoginCount  int        `gorm:"default:0" json:"failed_login_count"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`

	// 메타데이터 필드
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 관계 필드
	TokenGroups []TokenGroupModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"token_groups,omitempty"`
	Activities  []ActivityModel   `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"activities,omitempty"`
}

// TableName 테이블 이름 지정
func (UserModel) TableName() string {
	return "users"
}
