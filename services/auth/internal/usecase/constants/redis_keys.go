package constants

import "time"

// Redis 키 관련 상수
const (
	// TokenKeyPrefix 인증 토큰 키 접두사
	TokenKeyPrefix = "token:"

	// EmailKeyPrefix 이메일 인증 키 접두사
	EmailKeyPrefix = "email:verify:"

	// MagicCodePrefix 매직 코드 키 접두사
	MagicCodePrefix = "magic:"

	// RevokedTokenPrefix 취소된 토큰 키 접두사
	RevokedTokenPrefix = "revoked_token:"

	// VerificationTokenExpiry 이메일 인증 토큰 만료 시간 (시간)
	VerificationTokenExpiry = 24

	// MagicCodeExpiry 매직 코드 만료 시간 (분)
	MagicCodeExpiry = 5

	// AccessTokenExpiry 액세스 토큰 만료 시간 (분)
	AccessTokenExpiry = 30

	// RefreshTokenExpiry 리프레시 토큰 만료 시간 (일)
	RefreshTokenExpiry = 30

	// SessionPrefix 세션 Redis 접두사
	SessionPrefix = "session:"

	// OTPCodePrefix OTP 코드 Redis 접두사
	OTPCodePrefix = "otp_code:"

	// DefaultExpiry 기본 만료 시간 (24시간)
	DefaultExpiry = 24 * time.Hour

	// SessionExpiry 세션 만료 시간 (24시간)
	SessionExpiry = 24

	// OTPExpiry OTP 코드 만료 시간 (5분)
	OTPExpiry = 5 * time.Minute
)

// 이메일 템플릿 유형
const (
	// EmailTemplateWelcome 환영 이메일
	EmailTemplateWelcome = "welcome"

	// EmailTemplateVerification 인증 이메일
	EmailTemplateVerification = "verification"

	// EmailTemplatePasswordReset 비밀번호 재설정 이메일
	EmailTemplatePasswordReset = "password_reset"
)
