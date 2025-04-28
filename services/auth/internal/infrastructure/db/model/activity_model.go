package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActivityModel 사용자 로그인 세션 트래킹을 위한 데이터베이스 모델
type ActivityModel struct {
	SessionID    string     `gorm:"type:char(52);primaryKey" json:"session_id"` // 세션 고유 식별자
	UserID       string     `gorm:"type:char(12);primaryKey" json:"user_id"`    // 연결된 사용자
	TokenGroupID uint       `gorm:"index" json:"token_group_id,omitempty"`      // 연결된 토큰 그룹
	IP           string     `gorm:"size:250" json:"ip"`                         // 출발지 IP 주소
	UserAgent    string     `gorm:"size:250" json:"useragent"`                  // 사용자 에이전트 정보
	DeviceUID    *uuid.UUID `gorm:"type:char(36);" json:"device_uid"`           // 기기 고유 식별자
	LoginAt      time.Time  `json:"login_at"`                                   // 세션 시작 시간
	LogoutAt     *time.Time `json:"logout_at,omitempty"`                        // 세션 종료 시간 (nil = 활성)
	LocationInfo string     `gorm:"size:250" json:"location_info,omitempty"`    // 위치 정보
	DeviceInfo   string     `gorm:"size:250" json:"device_info,omitempty"`      // 기기 정보

	// 메타데이터 필드
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 테이블 이름 지정
func (ActivityModel) TableName() string {
	return "login_activities"
}
