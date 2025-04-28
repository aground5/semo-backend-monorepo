package service

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
)

// TokenService 토큰 관련 도메인 로직을 처리하는 서비스
type TokenService struct {
	tokenRepository      repository.TokenRepository
	tokenGroupRepository repository.TokenGroupRepository
}

// NewTokenService 토큰 서비스 생성
func NewTokenService(
	tokenRepo repository.TokenRepository,
	tokenGroupRepo repository.TokenGroupRepository,
) *TokenService {
	return &TokenService{
		tokenRepository:      tokenRepo,
		tokenGroupRepository: tokenGroupRepo,
	}
}

// GenerateTokens 사용자 ID로 토큰 그룹 및 토큰 생성
func (s *TokenService) GenerateTokens(ctx context.Context, userID, deviceName, deviceInfo string, accessTokenData, refreshTokenData entity.TokenData) (*entity.TokenGroup, *entity.Token, *entity.Token, error) {
	// 1. 토큰 그룹 생성
	tokenGroup := entity.NewTokenGroup(userID, deviceName, deviceInfo)

	if err := s.tokenGroupRepository.Create(ctx, tokenGroup); err != nil {
		return nil, nil, nil, err
	}

	// 2. 액세스 토큰 생성
	accessToken := entity.NewToken(
		tokenGroup.ID,
		accessTokenData.TokenValue,
		"access",
		accessTokenData.ExpiresAt,
	)

	if err := s.tokenRepository.Create(ctx, accessToken); err != nil {
		return nil, nil, nil, err
	}

	// 3. 리프레시 토큰 생성
	refreshToken := entity.NewToken(
		tokenGroup.ID,
		refreshTokenData.TokenValue,
		"refresh",
		refreshTokenData.ExpiresAt,
	)

	if err := s.tokenRepository.Create(ctx, refreshToken); err != nil {
		return nil, nil, nil, err
	}

	return tokenGroup, accessToken, refreshToken, nil
}

// InvalidateTokenGroup 토큰 그룹 및 관련 토큰 무효화
func (s *TokenService) InvalidateTokenGroup(ctx context.Context, groupID uint) error {
	// 1. 그룹에 속한 모든 토큰 조회
	tokens, err := s.tokenRepository.FindByGroupID(ctx, groupID)
	if err != nil {
		return err
	}

	// 2. 모든 토큰 삭제
	for _, token := range tokens {
		if err := s.tokenRepository.Delete(ctx, token.ID); err != nil {
			return err
		}
	}

	// 3. 토큰 그룹 삭제
	return s.tokenGroupRepository.Delete(ctx, groupID)
}

// InvalidateUserTokens 사용자의 모든 토큰 무효화
func (s *TokenService) InvalidateUserTokens(ctx context.Context, userID string) error {
	// 1. 사용자의 모든 토큰 그룹 조회
	tokenGroups, err := s.tokenGroupRepository.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// 2. 각 토큰 그룹 무효화
	for _, group := range tokenGroups {
		if err := s.InvalidateTokenGroup(ctx, group.ID); err != nil {
			return err
		}
	}

	return nil
}
