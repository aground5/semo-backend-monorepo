package constants

// Redis 키 관련 상수
const (
	// TokenKeyPrefix 인증 토큰 키 접두사
	TokenKeyPrefix = "token:"

	// EmailKeyPrefix 이메일 인증 키 접두사
	EmailKeyPrefix = "email:verify:"

	// MagicCodePrefix 매직 코드 키 접두사
	MagicCodePrefix = "magic:"

	// RevokedTokenPrefix 취소된 토큰 키 접두사
	RevokedTokenPrefix = "at:revoked:"

	// VerificationTokenExpiry 이메일 인증 토큰 만료 시간 (시간)
	VerificationTokenExpiry = 24

	// MagicCodeExpiry 매직 코드 만료 시간 (분)
	MagicCodeExpiry = 5

	// AccessTokenExpiry 액세스 토큰 만료 시간 (분)
	AccessTokenExpiry = 15

	// RefreshTokenExpiry 리프레시 토큰 만료 시간 (일)
	RefreshTokenExpiry = 30
)
