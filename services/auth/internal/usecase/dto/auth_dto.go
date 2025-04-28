package dto

import (
	"time"

	"github.com/google/uuid"
)

// RegisterParams 사용자 회원가입 매개변수
type RegisterParams struct {
	Email     string
	Password  string
	Username  string
	IP        string
	UserAgent string
}

// LoginParams 로그인 매개변수
type LoginParams struct {
	Email     string
	Password  string
	Code      string // 매직 코드 로그인용
	IP        string
	UserAgent string
	SessionID string
	DeviceUID *uuid.UUID
}

// DeviceInfo 장치 정보
type DeviceInfo struct {
	DeviceUID *uuid.UUID
	IP        string
	UserAgent string
	SessionID string
}

// AuthTokens 인증 토큰 응답
type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}
