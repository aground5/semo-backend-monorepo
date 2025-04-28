package init

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// UseCases 애플리케이션의 모든 유스케이스 컨테이너
type UseCases struct {
	AuthUseCase    interfaces.AuthUseCase
	SessionUseCase *usecase.SessionUseCase
	UserUseCase    *usecase.UserUseCase
	TokenUseCase   interfaces.TokenUseCase
	OTPUseCase     interfaces.OTPUseCase
	EmailUseCase   interfaces.EmailUseCase
}

// NewUseCases 모든 유스케이스 인스턴스 생성 및 초기화
func NewUseCases(
	repos *repository.Repositories,
	logger *zap.Logger,
) *UseCases {
	// 1. 프록시로 사용할 유스케이스 인스턴스 생성
	useCases := &UseCases{}

	// 2. 먼저 하위 유스케이스 초기화

	// 토큰 유스케이스 초기화
	useCases.TokenUseCase = usecase.NewTokenUseCase(
		logger,
		repos.Token,
	)

	// OTP 유스케이스 초기화
	useCases.OTPUseCase = usecase.NewOTPUseCase(
		logger,
		repos.Cache,
	)

	// 이메일 유스케이스 초기화
	useCases.EmailUseCase = usecase.NewEmailUseCase(
		logger,
		repos.Mail,
	)

	// 사용자 유스케이스 초기화
	useCases.UserUseCase = usecase.NewUserUseCase(
		logger,
		repos.User,
		repos.Token,
		repos.AuditLog,
	)

	// 세션 유스케이스 초기화
	useCases.SessionUseCase = usecase.NewSessionUseCase(
		logger,
		repos.User,
		repos.Token,
		repos.AuditLog,
		useCases.TokenUseCase,
		useCases.OTPUseCase,
	)

	// 인증 유스케이스 초기화
	authUC := usecase.NewAuthUseCase(
		logger,
		repos.User,
		repos.AuditLog,
		useCases.EmailUseCase,
	)

	// 3. Facade 패턴을 통한 통합 인증 유스케이스 생성
	useCases.AuthUseCase = NewAuthUseCaseFacade(
		authUC,
		useCases.SessionUseCase,
	)

	return useCases
}

// AuthUseCaseFacade 인터페이스 구현을 위한 퍼사드 패턴 구현체
type AuthUseCaseFacade struct {
	authUC    *usecase.AuthUseCase
	sessionUC *usecase.SessionUseCase
}

// NewAuthUseCaseFacade 인증 퍼사드 생성
func NewAuthUseCaseFacade(
	authUC *usecase.AuthUseCase,
	sessionUC *usecase.SessionUseCase,
) interfaces.AuthUseCase {
	return &AuthUseCaseFacade{
		authUC:    authUC,
		sessionUC: sessionUC,
	}
}

// 인터페이스 구현 메서드들

// Register 사용자 회원가입
func (f *AuthUseCaseFacade) Register(ctx context.Context, params dto.RegisterParams) (*entity.User, error) {
	return f.authUC.Register(ctx, params)
}

// VerifyEmail 이메일 인증
func (f *AuthUseCaseFacade) VerifyEmail(ctx context.Context, token string) (*entity.User, error) {
	return f.authUC.VerifyEmail(ctx, token)
}

// ResendVerificationEmail 이메일 인증 재발송
func (f *AuthUseCaseFacade) ResendVerificationEmail(ctx context.Context, email string) error {
	return f.authUC.ResendVerificationEmail(ctx, email)
}

// Login 사용자 로그인
func (f *AuthUseCaseFacade) Login(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	return f.sessionUC.Login(ctx, params)
}

// LoginWithPassword 비밀번호로 로그인
func (f *AuthUseCaseFacade) LoginWithPassword(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	return f.sessionUC.LoginWithPassword(ctx, params)
}

// AutoLogin 자동 로그인 (리프레시 토큰 사용)
func (f *AuthUseCaseFacade) AutoLogin(ctx context.Context, email, refreshToken string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error) {
	return f.sessionUC.AutoLogin(ctx, email, refreshToken, deviceInfo)
}

// Logout 로그아웃
func (f *AuthUseCaseFacade) Logout(ctx context.Context, sessionID, accessToken, refreshToken, userID string) error {
	return f.sessionUC.Logout(ctx, sessionID, accessToken, refreshToken, userID)
}

// RefreshTokens 토큰 갱신
func (f *AuthUseCaseFacade) RefreshTokens(ctx context.Context, refreshToken, userID, sessionID string) (*dto.AuthTokens, error) {
	return f.sessionUC.RefreshTokens(ctx, refreshToken, userID, sessionID)
}

// GenerateTokensAfter2FA 2FA 인증 후 토큰 생성
func (f *AuthUseCaseFacade) GenerateTokensAfter2FA(ctx context.Context, userID string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error) {
	return f.sessionUC.GenerateTokensAfter2FA(ctx, userID, deviceInfo)
}
