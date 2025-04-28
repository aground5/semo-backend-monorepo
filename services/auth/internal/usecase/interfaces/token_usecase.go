package interfaces

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/dto"
)

// TokenUseCase 토큰 관련 유스케이스 인터페이스
type TokenUseCase interface {
	// FindOrCreateTokenGroup 토큰 그룹 찾기 또는 생성
	FindOrCreateTokenGroup(ctx context.Context, userID string) (*entity.TokenGroup, error)

	// GenerateAccessToken 사용자 정보로부터 액세스 토큰 생성
	GenerateAccessToken(ctx context.Context, user *entity.User) (string, error)

	// GenerateRefreshToken 리프레시 토큰 생성
	GenerateRefreshToken(ctx context.Context, tokenGroupID uint) (string, *entity.Token, error)

	// ValidateAndRegenerateRefreshToken 리프레시 토큰 검증 및 갱신
	ValidateAndRegenerateRefreshToken(ctx context.Context, refreshToken string, user *entity.User, sessionID string) (uint, *entity.User, string, error)

	// ValidateAccessToken 액세스 토큰 검증
	ValidateAccessToken(ctx context.Context, accessToken string) (*entity.User, error)

	// RegenerateRefreshToken 리프레시 토큰 갱신
	RegenerateRefreshToken(ctx context.Context, refreshToken string) (string, *entity.Token, error)

	// RegenerateAccessToken 액세스 토큰 갱신
	RegenerateAccessToken(ctx context.Context, accessToken string) (string, *entity.Token, error)

	// RevokeTokenGroup 토큰 그룹 폐기
	RevokeTokenGroup(ctx context.Context, tokenGroupID uint) error

	// RevokeRefreshToken 리프레시 토큰 폐기
	RevokeRefreshToken(ctx context.Context, refreshToken string) error

	// RevokeAccessToken 액세스 토큰 폐기
	RevokeAccessToken(ctx context.Context, accessToken string) error

	// RefreshTokens 토큰 갱신
	RefreshTokens(ctx context.Context, refreshToken, userID, sessionID string) (*dto.AuthTokens, error)
}
