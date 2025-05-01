package models

// AuditLogType defines the types of auditable events in the system
// Used for categorizing and filtering audit logs
type AuditLogType string

const (
	// Authentication-related audit log types
	AuditLogTypeLoginSuccess        AuditLogType = "LOGIN_SUCCESS"         // User successfully logged in
	AuditLogTypeLoginFailed         AuditLogType = "LOGIN_FAILED"          // Failed login attempt
	AuditLogTypeLogoutSuccess       AuditLogType = "LOGOUT_SUCCESS"        // User successfully logged out
	AuditLogTypeAutoLoginSuccess    AuditLogType = "AUTO_LOGIN_SUCCESS"    // Automatic login succeeded
	AuditLogTypeRefreshTokenSuccess AuditLogType = "REFRESH_TOKEN_SUCCESS" // Token refresh succeeded
	AuditLogTypeRefreshTokenInvalid AuditLogType = "REFRESH_TOKEN_INVALID" // Token refresh failed

	// User management audit log types
	AuditLogTypeUserRegistered     AuditLogType = "USER_REGISTERED"      // New user registered
	AuditLogTypeEmailVerified      AuditLogType = "EMAIL_VERIFIED"       // User verified their email
	AuditLogTypeMagicCodeGenerated AuditLogType = "MAGIC_CODE_GENERATED" // Magic code was generated

	// Device management audit log types
	AuditLogTypeNewDeviceRegistered AuditLogType = "NEW_DEVICE_REGISTERED" // New device was registered

	// Two-factor authentication audit log types
	AuditLogType2FAEnabled  AuditLogType = "2FA_ENABLED"  // 2FA was enabled
	AuditLogType2FADisabled AuditLogType = "2FA_DISABLED" // 2FA was disabled
	AuditLogType2FAVerified AuditLogType = "2FA_VERIFIED" // 2FA code verified successfully

	// Security-related audit log types
	AuditLogTypeSecurityAlert  AuditLogType = "SECURITY_ALERT"   // Security alert generated
	AuditLogTypeBlockedIPLogin AuditLogType = "BLOCKED_IP_LOGIN" // Login attempt from blocked IP
	AuditLogTypeHoneypotAccess AuditLogType = "HONEYPOT_ACCESS"  // Honeypot account accessed
)
