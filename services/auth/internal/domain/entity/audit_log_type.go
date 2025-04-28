package entity

// AuditLogType 시스템에서 감사 가능한 이벤트 유형을 정의합니다
// 감사 로그 분류 및 필터링에 사용됩니다
type AuditLogType string

const (
	// 인증 관련 감사 로그 유형
	AuditLogTypeLoginSuccess        AuditLogType = "LOGIN_SUCCESS"         // 사용자 로그인 성공
	AuditLogTypeLoginFailed         AuditLogType = "LOGIN_FAILED"          // 로그인 시도 실패
	AuditLogTypeLogoutSuccess       AuditLogType = "LOGOUT_SUCCESS"        // 사용자 로그아웃 성공
	AuditLogTypeAutoLoginSuccess    AuditLogType = "AUTO_LOGIN_SUCCESS"    // 자동 로그인 성공
	AuditLogTypeRefreshTokenSuccess AuditLogType = "REFRESH_TOKEN_SUCCESS" // 토큰 갱신 성공
	AuditLogTypeRefreshTokenInvalid AuditLogType = "REFRESH_TOKEN_INVALID" // 토큰 갱신 실패

	// 사용자 관리 감사 로그 유형
	AuditLogTypeUserRegistered     AuditLogType = "USER_REGISTERED"      // 신규 사용자 등록
	AuditLogTypeEmailVerified      AuditLogType = "EMAIL_VERIFIED"       // 사용자 이메일 인증
	AuditLogTypeMagicCodeGenerated AuditLogType = "MAGIC_CODE_GENERATED" // 매직 코드 생성

	// 기기 관리 감사 로그 유형
	AuditLogTypeNewDeviceRegistered AuditLogType = "NEW_DEVICE_REGISTERED" // 새 기기 등록

	// 2단계 인증 감사 로그 유형
	AuditLogType2FAEnabled  AuditLogType = "2FA_ENABLED"  // 2단계 인증 활성화
	AuditLogType2FADisabled AuditLogType = "2FA_DISABLED" // 2단계 인증 비활성화
	AuditLogType2FAVerified AuditLogType = "2FA_VERIFIED" // 2단계 인증 코드 확인 성공

	// 보안 관련 감사 로그 유형
	AuditLogTypeSecurityAlert  AuditLogType = "SECURITY_ALERT"   // 보안 경고 생성
	AuditLogTypeBlockedIPLogin AuditLogType = "BLOCKED_IP_LOGIN" // 차단된 IP에서 로그인 시도
	AuditLogTypeHoneypotAccess AuditLogType = "HONEYPOT_ACCESS"  // 허니팟 계정 접근

	// 다양한 감사 로그 유형 상수 정의
	AuditLogTypeUserRegistration  AuditLogType = "USER_REGISTRATION"
	AuditLogTypePasswordReset     AuditLogType = "PASSWORD_RESET"
	AuditLogTypeUserProfileUpdate AuditLogType = "USER_PROFILE_UPDATE"
	AuditLogTypePermissionChange  AuditLogType = "PERMISSION_CHANGE"
	AuditLogTypeAPIRequest        AuditLogType = "API_REQUEST"
	AuditLogTypeDataExport        AuditLogType = "DATA_EXPORT"
	AuditLogTypeDataImport        AuditLogType = "DATA_IMPORT"
	AuditLogTypeUserDeletion      AuditLogType = "USER_DELETION"
	AuditLogTypeAdminAction       AuditLogType = "ADMIN_ACTION"
	AuditLogTypeMailRequest       AuditLogType = "MAIL_REQUEST"
	AuditLogTypeEmailVerification AuditLogType = "EMAIL_VERIFICATION"
	AuditLogTypePhoneVerification AuditLogType = "PHONE_VERIFICATION"
	AuditLogTypeTokenCreation     AuditLogType = "TOKEN_CREATION"
	AuditLogTypeTokenRevocation   AuditLogType = "TOKEN_REVOCATION"
	AuditLogTypeRoleAssignment    AuditLogType = "ROLE_ASSIGNMENT"
	AuditLogTypeTwoFactorAuth     AuditLogType = "TWO_FACTOR_AUTH"
	AuditLogTypeUserStatusChange  AuditLogType = "USER_STATUS_CHANGE"
)
