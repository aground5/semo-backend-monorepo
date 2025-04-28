package entity

import (
	"time"
)

// TokenGroup 사용자 인증 세션을 위한 토큰 그룹 도메인 엔티티
type TokenGroup struct {
	ID        uint
	UserID    string    // 연결된 사용자
	Name      string    // 토큰 그룹 이름/설명
	Device    string    // 기기 정보
	CreatedAt time.Time // 생성 시간
}

// NewTokenGroup 새 토큰 그룹 생성
func NewTokenGroup(userID, name, device string) *TokenGroup {
	return &TokenGroup{
		UserID:    userID,
		Name:      name,
		Device:    device,
		CreatedAt: time.Now(),
	}
}

// UpdateDevice 기기 정보 업데이트
func (tg *TokenGroup) UpdateDevice(device string) {
	tg.Device = device
}

// UpdateName 이름 업데이트
func (tg *TokenGroup) UpdateName(name string) {
	tg.Name = name
}
