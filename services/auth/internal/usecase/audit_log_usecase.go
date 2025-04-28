package usecase

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// AuditLogUseCase 감사 로그 유스케이스 구현체
type AuditLogUseCase struct {
	logger          *zap.Logger
	auditRepository repository.AuditLogRepository
}

// NewAuditLogUseCase 새 감사 로그 유스케이스 생성
func NewAuditLogUseCase(
	logger *zap.Logger,
	auditRepo repository.AuditLogRepository,
) interfaces.AuditLogUseCase {
	return &AuditLogUseCase{
		logger:          logger,
		auditRepository: auditRepo,
	}
}

// AddLog 감사 로그 추가
func (uc *AuditLogUseCase) AddLog(ctx context.Context, logType entity.AuditLogType, content map[string]interface{}, userID *string) error {
	// 새 감사 로그 엔티티 생성
	auditLog := entity.NewAuditLog(userID, logType, content)

	// 로그 저장
	if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
		uc.logger.Error("감사 로그 저장 실패",
			zap.String("type", string(logType)),
			zap.Any("content", content),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// GetUserLogs 특정 사용자의 감사 로그 조회
func (uc *AuditLogUseCase) GetUserLogs(ctx context.Context, userID string, page, limit int) ([]*entity.AuditLog, error) {
	// 사용자 ID로 필터링
	filter := map[string]interface{}{
		"user_id": userID,
	}

	// 로그 조회
	logs, err := uc.auditRepository.FindByFilter(ctx, filter, page, limit)
	if err != nil {
		uc.logger.Error("사용자 감사 로그 조회 실패",
			zap.String("userID", userID),
			zap.Error(err),
		)
		return nil, err
	}

	return logs, nil
}

// GetLogs 감사 로그 조회 (필터링 가능)
func (uc *AuditLogUseCase) GetLogs(ctx context.Context, filter map[string]interface{}, page, limit int) ([]*entity.AuditLog, error) {
	// 로그 조회
	logs, err := uc.auditRepository.FindByFilter(ctx, filter, page, limit)
	if err != nil {
		uc.logger.Error("감사 로그 조회 실패",
			zap.Any("filter", filter),
			zap.Error(err),
		)
		return nil, err
	}

	return logs, nil
}
