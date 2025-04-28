package repository

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
)

// UserRepository 사용자 엔티티 관련 저장소 인터페이스
type UserRepository interface {
	// FindByID ID로 사용자 조회
	FindByID(ctx context.Context, id string) (*entity.User, error)

	// FindByEmail 이메일로 사용자 조회
	FindByEmail(ctx context.Context, email string) (*entity.User, error)

	// FindByUsername 사용자명으로 사용자 조회
	FindByUsername(ctx context.Context, username string) (*entity.User, error)

	// Create 새 사용자 생성
	Create(ctx context.Context, user *entity.User) error

	// Update 사용자 정보 업데이트
	Update(ctx context.Context, user *entity.User) error

	// Delete 사용자 삭제
	Delete(ctx context.Context, id string) error
}
