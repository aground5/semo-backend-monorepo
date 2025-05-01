package auth

import (
	"authn-server/internal/models"
	"context"
	"github.com/google/uuid"
	"time"
)

// AuthServiceInterface는 인증 서비스의 공개 API를 정의합니다.
type AuthServiceInterface interface {
	// 기본 인증 기능
	Register(ctx context.Context, params RegisterParams) (*models.User, error)
	VerifyEmail(token string) (*models.User, error)
	ResendVerificationEmail(email string) error
	Login(ctx context.Context, params LoginParams) (*AuthTokens, *models.User, error)
	LoginWithPassword(ctx context.Context, params LoginParams) (*AuthTokens, *models.User, error)
	AutoLogin(ctx context.Context, email, refreshToken string, deviceInfo DeviceInfo) (*AuthTokens, error)
	Logout(ctx context.Context, sessionID, accessToken, refreshToken, userID string) error
	RefreshTokens(ctx context.Context, refreshToken, userID, sessionID string) (*AuthTokens, error)

	// 2FA 관련 기능
	GenerateTokensAfter2FA(ctx context.Context, userID string, deviceInfo DeviceInfo) (*AuthTokens, error)

	// OTP 관련 기능
	SendMagicCode(ctx context.Context, email, ip, userAgent string) error
}

// TokenManagerInterface는 토큰 관리 기능을 정의합니다.
type TokenManagerInterface interface {
	GenerateAccessToken(user *models.User) (string, error)
	GenerateRefreshToken(groupID uint) (string, *models.Token)
	ValidateRefreshToken(refreshToken string) (uint, *models.User, string, error)
	RevokeTokenGroup(tokenGroupID uint) error
	RevokeAccessToken(accessToken string) error
}

// EmailVerifierInterface는 이메일 인증 기능을 정의합니다.
type EmailVerifierInterface interface {
	GenerateVerificationToken(email string) (string, error)
	VerifyToken(token string) (string, error)
	SendVerificationEmail(email, name, token string) error
	DeleteTokens(token, email string) error
}

// OtpServiceInterface는 일회용 코드 기능을 정의합니다.
type OtpServiceInterface interface {
	SendMagicCode(email, ip, userAgent string) error
	VerifyMagicCode(email, code string) (bool, error)
}

// 요청 매개변수 구조체들
type RegisterParams struct {
	Email     string
	Password  string
	Username  string
	IP        string
	UserAgent string
}

type LoginParams struct {
	Email     string
	Password  string
	Code      string // 매직 코드 로그인용
	IP        string
	UserAgent string
	SessionID string
	DeviceUID *uuid.UUID
}

// 장치 정보 구조체
type DeviceInfo struct {
	DeviceUID *uuid.UUID
	IP        string
	UserAgent string
	SessionID string
}

// 인증 토큰 응답 구조체
type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}
