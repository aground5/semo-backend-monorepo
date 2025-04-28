package interfaces

import (
	"context"
)

// EmailUseCase 이메일 인증 관련 유스케이스 인터페이스
type EmailUseCase interface {
	// GenerateVerificationToken 이메일 인증 토큰 생성
	GenerateVerificationToken(ctx context.Context, email string) (string, error)

	// VerifyToken 인증 토큰 확인
	VerifyToken(ctx context.Context, token string) (string, error)

	// SendVerificationEmail 인증 이메일 발송
	SendVerificationEmail(ctx context.Context, email, name, token string) error

	// DeleteTokens 인증 완료 후 토큰 삭제
	DeleteTokens(ctx context.Context, token, email string) error
}
