package repository

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/domain/entity"
)

// ItemRepository 아이템 엔티티 관련 저장소 인터페이스
type ItemRepository interface {
	// FindByID ID로 아이템 조회
	FindByID(ctx context.Context, id string) (*entity.Item, error)

	// FindByParentID 부모 ID로 자식 아이템 목록 조회
	FindByParentID(ctx context.Context, parentID string, limit, offset int) ([]*entity.Item, error)

	// FindRootItems 루트 아이템 목록 조회 (부모가 없는 아이템)
	FindRootItems(ctx context.Context, createdBy string, limit, offset int) ([]*entity.Item, error)

	// Create 새 아이템 생성
	Create(ctx context.Context, item *entity.Item) error

	// Update 아이템 정보 업데이트
	Update(ctx context.Context, item *entity.Item) error

	// Delete 아이템 삭제
	Delete(ctx context.Context, id string) error

	// CountByParentID 부모 ID로 자식 아이템 개수 조회
	CountByParentID(ctx context.Context, parentID string) (int64, error)

	// CountRootItems 루트 아이템 개수 조회
	CountRootItems(ctx context.Context, createdBy string) (int64, error)

	// UpdatePosition 아이템 위치 업데이트
	UpdatePosition(ctx context.Context, id string, position float64) error

	// FindProjectsByUser 사용자가 생성한 프로젝트 목록 조회
	FindProjectsByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.Item, error)

	// FindTasksByProject 프로젝트에 속한 태스크 목록 조회
	FindTasksByProject(ctx context.Context, projectID string, limit, offset int) ([]*entity.Item, error)

	// FindDependencies 아이템의 의존성 목록 조회
	FindDependencies(ctx context.Context, itemID string) ([]*entity.Item, error)

	// AddDependency 아이템에 의존성 추가
	AddDependency(ctx context.Context, itemID, dependencyID string) error

	// RemoveDependency 아이템에서 의존성 제거
	RemoveDependency(ctx context.Context, itemID, dependencyID string) error
}
