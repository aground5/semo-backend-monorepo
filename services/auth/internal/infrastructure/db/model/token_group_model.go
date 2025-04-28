package model

import (
	"time"

	"gorm.io/gorm"
)

// TokenGroupModel 토큰 그룹 데이터베이스 모델
type TokenGroupModel struct {
	ID     uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID string `gorm:"type:char(12);not null;index" json:"user_id"` // 연결된 사용자
	Name   string `gorm:"size:100" json:"name,omitempty"`              // 토큰 그룹 이름/설명
	Device string `gorm:"size:250" json:"device,omitempty"`            // 기기 정보

	// 메타데이터 필드
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 관계 필드
	Tokens []TokenModel `gorm:"foreignKey:GroupID" json:"tokens,omitempty"`
}

// TableName 테이블 이름 지정
func (TokenGroupModel) TableName() string {
	return "token_groups"
}
