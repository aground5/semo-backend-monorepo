package dto

import (
	"time"
)

// TokenData JWT 토큰 생성에 필요한 데이터
type TokenData struct {
	TokenValue string    // 토큰 값
	ExpiresAt  time.Time // 만료 시간
}

// JWTClaims JWT 토큰에 포함되는 클레임 정보
type JWTClaims struct {
	Subject  string    // 사용자 ID
	Name     string    // 사용자 이름
	Email    string    // 사용자 이메일
	Issuer   string    // 발급자
	IssuedAt time.Time // 발급 시간
	ExpireAt time.Time // 만료 시간
}

// TokenPair 액세스 토큰과 리프레시 토큰 쌍
type TokenPair struct {
	AccessToken  string    // 액세스 토큰
	RefreshToken string    // 리프레시 토큰
	ExpiresAt    time.Time // 액세스 토큰 만료 시간
}

// TokenValidationResult 토큰 검증 결과
type TokenValidationResult struct {
	Valid     bool   // 토큰 유효성
	UserID    string // 사용자 ID
	TokenID   uint   // 토큰 ID
	GroupID   uint   // 토큰 그룹 ID
	TokenType string // 토큰 유형
}
