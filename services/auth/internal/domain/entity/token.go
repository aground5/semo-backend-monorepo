package entity

import (
	"time"
)

// Token 사용자 인증 토큰 도메인 엔티티
type Token struct {
	ID        uint
	GroupID   uint      // 연결된 토큰 그룹
	Token     string    // 암호화된 토큰 값
	TokenType string    // 토큰 유형 (access, refresh)
	ExpiresAt time.Time // 만료 시간
}

// NewToken 새 토큰 생성
func NewToken(groupID uint, token, tokenType string, expiresAt time.Time) *Token {
	return &Token{
		GroupID:   groupID,
		Token:     token,
		TokenType: tokenType,
		ExpiresAt: expiresAt,
	}
}

// IsExpired 토큰이 만료되었는지 확인
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsAccess 접근 토큰인지 확인
func (t *Token) IsAccess() bool {
	return t.TokenType == "access"
}

// IsRefresh 갱신 토큰인지 확인
func (t *Token) IsRefresh() bool {
	return t.TokenType == "refresh"
}
