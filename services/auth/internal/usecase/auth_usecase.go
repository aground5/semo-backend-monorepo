package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/service"
)

// AuthUseCase 인증 관련 유스케이스
type AuthUseCase struct {
	userRepository repository.UserRepository
	tokenService   *service.TokenService
}

// NewAuthUseCase 인증 유스케이스 생성
func NewAuthUseCase(
	userRepo repository.UserRepository,
	tokenService *service.TokenService,
) *AuthUseCase {
	return &AuthUseCase{
		userRepository: userRepo,
		tokenService:   tokenService,
	}
}

// LoginInput 로그인 입력 데이터
type LoginInput struct {
	Email      string
	Password   string
	IP         string
	UserAgent  string
	DeviceInfo string
}

// LoginOutput 로그인 출력 데이터
type LoginOutput struct {
	User         *entity.User
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// Login 사용자 로그인 처리
func (uc *AuthUseCase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// 1. 이메일로 사용자 조회
	user, err := uc.userRepository.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("사용자를 찾을 수 없습니다")
	}

	// 2. 비밀번호 검증 (실제 구현에서는 해시 비교 로직 필요)
	// 여기서는 간단히 구현
	if user.Password != input.Password {
		user.IncrementFailedLogin()
		if err := uc.userRepository.Update(ctx, user); err != nil {
			return nil, err
		}
		return nil, errors.New("비밀번호가 일치하지 않습니다")
	}

	// 3. 사용자 상태 확인
	if !user.IsActive() {
		return nil, errors.New("계정이 활성 상태가 아닙니다")
	}

	// 4. 토큰 생성 데이터 준비
	accessTokenExpiry := time.Now().Add(15 * time.Minute)
	refreshTokenExpiry := time.Now().Add(24 * time.Hour * 30) // 30일

	accessTokenData := entity.NewTokenData("access_token_value", accessTokenExpiry)
	refreshTokenData := entity.NewTokenData("refresh_token_value", refreshTokenExpiry)

	// 5. 토큰 그룹 및 토큰 생성
	_, accessToken, refreshToken, err := uc.tokenService.GenerateTokens(
		ctx,
		user.ID,
		"Device Name",
		input.DeviceInfo,
		accessTokenData,
		refreshTokenData,
	)
	if err != nil {
		return nil, err
	}

	// 6. 로그인 성공 기록
	user.RecordLogin(input.IP)
	if err := uc.userRepository.Update(ctx, user); err != nil {
		return nil, err
	}

	// 7. 응답 생성
	return &LoginOutput{
		User:         user,
		AccessToken:  accessToken.Token,
		RefreshToken: refreshToken.Token,
		ExpiresAt:    accessTokenExpiry,
	}, nil
}

// LogoutInput 로그아웃 입력 데이터
type LogoutInput struct {
	UserID    string
	TokenID   uint
	GroupID   uint
	LogoutAll bool
}

// Logout 사용자 로그아웃 처리
func (uc *AuthUseCase) Logout(ctx context.Context, input LogoutInput) error {
	if input.LogoutAll {
		// 모든 기기에서 로그아웃
		return uc.tokenService.InvalidateUserTokens(ctx, input.UserID)
	} else {
		// 현재 기기에서만 로그아웃
		return uc.tokenService.InvalidateTokenGroup(ctx, input.GroupID)
	}
}
