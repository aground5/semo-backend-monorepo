package interfaces

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
)

// AuditLogUseCase 감사 로그 기능을 위한 인터페이스
type AuditLogUseCase interface {
	// AddLog 감사 로그 추가
	AddLog(ctx context.Context, logType entity.AuditLogType, content map[string]interface{}, userID *string) error

	// GetUserLogs 특정 사용자의 감사 로그 조회
	GetUserLogs(ctx context.Context, userID string, page, limit int) ([]*entity.AuditLog, error)
}
