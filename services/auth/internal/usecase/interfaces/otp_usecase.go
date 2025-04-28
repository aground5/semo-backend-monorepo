package interfaces

import (
	"context"
)

// OTPUseCase OTP 관련 유스케이스 인터페이스
type OTPUseCase interface {
	// SendMagicCode 매직 코드 발송
	SendMagicCode(ctx context.Context, email, ip, userAgent string) error

	// VerifyMagicCode 매직 코드 검증
	VerifyMagicCode(ctx context.Context, email, code string) (bool, error)

	// GenerateRandomCode 랜덤 코드 생성
	GenerateRandomCode(length int) string
}
