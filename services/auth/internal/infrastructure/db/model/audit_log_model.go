package model

import (
	"time"

	"gorm.io/gorm"
)

// AuditLogModel 감사 로그 데이터베이스 모델
type AuditLogModel struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *string        `gorm:"type:char(12);index" json:"user_id,omitempty"` // 사용자 ID (선택 사항)
	Type      string         `gorm:"size:50;not null;index" json:"type"`           // 로그 유형
	Content   string         `gorm:"type:text" json:"content"`                     // JSON 형식 콘텐츠
	CreatedAt time.Time      `gorm:"autoCreateTime;index" json:"created_at"`       // 생성 시간
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`             // 업데이트 시간
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`            // 소프트 삭제
}

// TableName 테이블 이름 지정
func (AuditLogModel) TableName() string {
	return "audit_logs"
}
