package repository

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
)

// TokenRepository 토큰 관련 저장소 인터페이스
type TokenRepository interface {
	// FindByID ID로 토큰 조회
	FindByID(ctx context.Context, id uint) (*entity.Token, error)

	// FindByToken 토큰 값으로 조회
	FindByToken(ctx context.Context, token string) (*entity.Token, error)

	// FindByGroupID 그룹 ID로 토큰 목록 조회
	FindByGroupID(ctx context.Context, groupID uint) ([]*entity.Token, error)

	// Create 새 토큰 생성
	Create(ctx context.Context, token *entity.Token) error

	// Update 토큰 정보 업데이트
	Update(ctx context.Context, token *entity.Token) error

	// Delete 토큰 삭제
	Delete(ctx context.Context, id uint) error
}

// TokenGroupRepository 토큰 그룹 관련 저장소 인터페이스
type TokenGroupRepository interface {
	// FindByID ID로 토큰 그룹 조회
	FindByID(ctx context.Context, id uint) (*entity.TokenGroup, error)

	// FindByUserID 사용자 ID로 토큰 그룹 목록 조회
	FindByUserID(ctx context.Context, userID string) ([]*entity.TokenGroup, error)

	// Create 새 토큰 그룹 생성
	Create(ctx context.Context, tokenGroup *entity.TokenGroup) error

	// Update 토큰 그룹 정보 업데이트
	Update(ctx context.Context, tokenGroup *entity.TokenGroup) error

	// Delete 토큰 그룹 삭제
	Delete(ctx context.Context, id uint) error
}
