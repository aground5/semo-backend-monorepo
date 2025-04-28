package repository

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
)

// AuditLogRepository 감사 로그 저장소 인터페이스
type AuditLogRepository interface {
	// Create 새 감사 로그 생성
	Create(ctx context.Context, log *entity.AuditLog) error

	// GetByID ID로 감사 로그 조회
	GetByID(ctx context.Context, id uint) (*entity.AuditLog, error)

	// ListByUserID 사용자 ID로 감사 로그 목록 조회
	ListByUserID(ctx context.Context, userID string, page, limit int) ([]*entity.AuditLog, int64, error)

	// ListByType 로그 유형으로 감사 로그 목록 조회
	ListByType(ctx context.Context, logType entity.AuditLogType, page, limit int) ([]*entity.AuditLog, int64, error)

	// Search 검색 조건으로 감사 로그 조회
	Search(
		ctx context.Context,
		userID *string,
		logTypes []entity.AuditLogType,
		startDate, endDate *string,
		page, limit int,
	) ([]*entity.AuditLog, int64, error)

	// Delete 감사 로그 삭제
	Delete(ctx context.Context, id uint) error
}
