package auth

// Redis 키 관련 상수
const (
	TokenKeyPrefix     = "token:"
	EmailKeyPrefix     = "email:verify:"
	MagicCodePrefix    = "magic:"
	RevokedTokenPrefix = "at:revoked:"
)

// 감사 로그 유형
const (
	AuditLogTypeUserRegistered    = "USER_REGISTERED"
	AuditLogTypeEmailVerified     = "EMAIL_VERIFIED"
	AuditLogTypeLoginSuccess      = "LOGIN_SUCCESS"
	AuditLogTypeLoginFailed       = "LOGIN_FAILED"
	AuditLogTypeMagicCodeGenerated = "MAGIC_CODE_GENERATED"
	AuditLogTypeLogoutSuccess     = "LOGOUT_SUCCESS"
	AuditLogTypeAutoLoginSuccess  = "AUTO_LOGIN_SUCCESS"
	AuditLogTypeRefreshTokenSuccess = "REFRESH_TOKEN_SUCCESS"
	AuditLogTypeRefreshTokenInvalid = "REFRESH_TOKEN_INVALID"
)

// 오류 코드 상수
const (
	ErrInvalidCredentials = "invalid_credentials"
	ErrEmailAlreadyExists = "email_already_exists"
	ErrInvalidToken       = "invalid_token"
	ErrEmailNotVerified   = "email_not_verified"
	ErrUserNotFound       = "user_not_found"
	ErrTokenExpired       = "token_expired"
	ErrSessionInvalid     = "session_invalid"
)