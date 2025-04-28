package interfaces

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
)

// TokenUseCase 토큰 관련 유스케이스 인터페이스
type TokenUseCase interface {
	// GenerateAccessToken 사용자 정보로부터 액세스 토큰 생성
	GenerateAccessToken(ctx context.Context, user *entity.User) (string, error)

	// GenerateRefreshToken 리프레시 토큰 생성
	GenerateRefreshToken(ctx context.Context, tokenGroupID uint) (string, *entity.Token, error)

	// ValidateRefreshToken 리프레시 토큰 검증
	ValidateRefreshToken(ctx context.Context, refreshToken string) (uint, *entity.User, string, error)

	// RevokeTokenGroup 토큰 그룹 폐기
	RevokeTokenGroup(ctx context.Context, tokenGroupID uint) error

	// RevokeAccessToken 액세스 토큰 폐기
	RevokeAccessToken(ctx context.Context, accessToken string) error

	// ValidateAccessToken 액세스 토큰 검증
	ValidateAccessToken(ctx context.Context, accessToken string) (*entity.User, error)
}
