package interfaces

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/dto"
)

// AuthUseCase 인증 관련 유스케이스 인터페이스
type AuthUseCase interface {
	// Register 사용자 회원가입
	Register(ctx context.Context, params dto.RegisterParams) (*entity.User, error)

	// VerifyEmail 이메일 인증
	VerifyEmail(ctx context.Context, token string) (*entity.User, error)

	// ResendVerificationEmail 이메일 인증 재발송
	ResendVerificationEmail(ctx context.Context, email string) error

	// Login 사용자 로그인
	Login(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error)

	// LoginWithPassword 비밀번호로 로그인
	LoginWithPassword(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error)

	// AutoLogin 자동 로그인 (리프레시 토큰 사용)
	AutoLogin(ctx context.Context, email, refreshToken string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error)

	// Logout 로그아웃
	Logout(ctx context.Context, sessionID, accessToken, refreshToken, userID string) error

	// RefreshTokens 토큰 갱신
	RefreshTokens(ctx context.Context, refreshToken, userID, sessionID string) (*dto.AuthTokens, error)

	// GenerateTokensAfter2FA 2FA 인증 후 토큰 생성
	GenerateTokensAfter2FA(ctx context.Context, userID string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error)
}
