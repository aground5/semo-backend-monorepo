package usecase

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// UseCases는 모든 유스케이스를 담고 있는 구조체입니다.
type UseCases struct {
	Auth     interfaces.AuthUseCase
	Token    interfaces.TokenUseCase
	Email    interfaces.EmailUseCase
	OTP      interfaces.OTPUseCase
	AuditLog interfaces.AuditLogUseCase
}

// SetupUseCases는 모든 유스케이스 구현체를 생성하고 의존성을 주입합니다.
func SetupUseCases(
	logger *zap.Logger,
	cfg *config.Config,
	repositories repository.Repositories,
) *UseCases {
	// 1. 기본 유스케이스 생성 (다른 유스케이스에 의존하지 않는 것부터)
	auditLogUC := NewAuditLogUseCase(
		logger,
		repositories.AuditLog,
	)

	// 2. 토큰 유스케이스 생성
	tokenConfig := TokenConfig{
		ServiceName:        cfg.Service.Name,
		JwtPrivateKey:      cfg.JWT.PrivateKey,
		JwtPublicKey:       cfg.JWT.PublicKey,
		AccessTokenExpiry:  cfg.JWT.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.JWT.RefreshTokenExpiry,
	}

	tokenUC := NewTokenUseCase(
		logger,
		tokenConfig,
		repositories.Token,
		repositories.User,
		repositories.Cache,
		repositories.AuditLog,
	)

	// 3. 이메일 유스케이스 생성
	emailUC := NewEmailUseCase(
		logger,
		repositories.Cache,
		repositories.Mail,
		repositories.User,
		repositories.AuditLog,
		cfg.Service.BaseURL,
		cfg.Email.SenderEmail,
	)

	// 4. OTP 유스케이스 생성
	otpUC := NewOTPUseCase(
		logger,
		repositories.Cache,
		repositories.Mail,
		repositories.AuditLog,
	)

	// 5. 인증 유스케이스 생성 (다른 유스케이스를 의존)
	authUC := NewAuthUseCase(
		logger,
		repositories.User,
		repositories.Token,
		repositories.AuditLog,
		tokenUC,
		otpUC,
		emailUC,
	)

	return &UseCases{
		Auth:     authUC,
		Token:    tokenUC,
		Email:    emailUC,
		OTP:      otpUC,
		AuditLog: auditLogUC,
	}
}
