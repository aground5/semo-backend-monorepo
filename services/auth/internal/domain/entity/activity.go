package entity

import (
	"time"
)

// Activity 사용자 로그인 세션 추적을 위한 도메인 엔티티
type Activity struct {
	SessionID    string     // 세션 고유 식별자
	UserID       string     // 연결된 사용자 ID
	TokenGroupID uint       // 연결된 토큰 그룹 ID
	IP           string     // 출발지 IP 주소
	UserAgent    string     // 사용자 에이전트 정보
	DeviceUID    string     // 기기 고유 식별자
	LoginAt      time.Time  // 세션 시작 시간
	LogoutAt     *time.Time // 세션 종료 시간 (nil = 활성)
	LocationInfo string     // 위치 정보
	DeviceInfo   string     // 기기 정보
}

// NewActivity 새 활동 기록 생성
func NewActivity(sessionID, userID string, tokenGroupID uint, ip, userAgent, deviceUID string) *Activity {
	return &Activity{
		SessionID:    sessionID,
		UserID:       userID,
		TokenGroupID: tokenGroupID,
		IP:           ip,
		UserAgent:    userAgent,
		DeviceUID:    deviceUID,
		LoginAt:      time.Now(),
	}
}

// IsActive 세션이 활성 상태인지 확인
func (a *Activity) IsActive() bool {
	return a.LogoutAt == nil
}

// EndSession 세션 종료 처리
func (a *Activity) EndSession() {
	now := time.Now()
	a.LogoutAt = &now
}

// UpdateLocationInfo 위치 정보 업데이트
func (a *Activity) UpdateLocationInfo(locationInfo string) {
	a.LocationInfo = locationInfo
}

// UpdateDeviceInfo 기기 정보 업데이트
func (a *Activity) UpdateDeviceInfo(deviceInfo string) {
	a.DeviceInfo = deviceInfo
}
