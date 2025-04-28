package init

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase"
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
