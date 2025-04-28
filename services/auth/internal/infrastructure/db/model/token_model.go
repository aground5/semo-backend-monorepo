package model

import (
	"time"

	"gorm.io/gorm"
)

// TokenModel 인증 토큰 데이터베이스 모델
type TokenModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   uint      `gorm:"index;not null" json:"group_id"`             // 연결된 토큰 그룹
	Token     string    `gorm:"size:250;not null" json:"token"`             // 암호화된 토큰 값
	TokenType string    `gorm:"size:50;default:'access'" json:"token_type"` // 토큰 유형 (access, refresh)
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`                 // 만료 시간

	// 메타데이터 필드
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 테이블 이름 지정
func (TokenModel) TableName() string {
	return "tokens"
}
