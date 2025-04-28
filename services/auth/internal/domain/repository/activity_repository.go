package repository

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
)

// ActivityRepository 활동 기록 관련 저장소 인터페이스
type ActivityRepository interface {
	// FindBySessionID 세션 ID로 활동 기록 조회
	FindBySessionID(ctx context.Context, sessionID string) (*entity.Activity, error)

	// FindByUserID 사용자 ID로 활동 기록 조회
	FindByUserID(ctx context.Context, userID string) ([]*entity.Activity, error)

	// FindActiveSessions 사용자의 활성 세션 목록 조회
	FindActiveSessions(ctx context.Context, userID string) ([]*entity.Activity, error)

	// Create 새 활동 기록 생성
	Create(ctx context.Context, activity *entity.Activity) error

	// Update 활동 기록 업데이트
	Update(ctx context.Context, activity *entity.Activity) error

	// Delete 활동 기록 삭제
	Delete(ctx context.Context, sessionID string) error
}
