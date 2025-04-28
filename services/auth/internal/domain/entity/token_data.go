package entity

import (
	"time"
)

// TokenData 토큰 생성에 필요한 데이터 구조체
type TokenData struct {
	TokenValue string    // 토큰 값
	ExpiresAt  time.Time // 만료 시간
}

// NewTokenData 새 토큰 데이터 생성
func NewTokenData(tokenValue string, expiresAt time.Time) TokenData {
	return TokenData{
		TokenValue: tokenValue,
		ExpiresAt:  expiresAt,
	}
}
