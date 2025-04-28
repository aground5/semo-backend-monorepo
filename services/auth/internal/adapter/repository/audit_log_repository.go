package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db/model"
	"gorm.io/gorm"
)

type AuditLogRepositoryImpl struct {
	db *gorm.DB
}

// NewAuditLogRepository 감사 로그 저장소 구현체 생성
func NewAuditLogRepository(db *gorm.DB) repository.AuditLogRepository {
	return &AuditLogRepositoryImpl{db: db}
}

// 도메인 엔티티를 DB 모델로 변환
func toAuditLogModel(auditLog *entity.AuditLog) (*model.AuditLogModel, error) {
	contentJSON, err := auditLog.ContentJSON()
	if err != nil {
		return nil, fmt.Errorf("JSON 직렬화 실패: %w", err)
	}

	return &model.AuditLogModel{
		ID:      auditLog.ID,
		UserID:  auditLog.UserID,
		Type:    string(auditLog.Type),
		Content: contentJSON,
	}, nil
}

// DB 모델을 도메인 엔티티로 변환
func toAuditLogEntity(auditLogModel *model.AuditLogModel) (*entity.AuditLog, error) {
	var content map[string]interface{}
	if auditLogModel.Content != "" {
		if err := json.Unmarshal([]byte(auditLogModel.Content), &content); err != nil {
			return nil, fmt.Errorf("JSON 역직렬화 실패: %w", err)
		}
	}

	return &entity.AuditLog{
		ID:        auditLogModel.ID,
		UserID:    auditLogModel.UserID,
		Type:      entity.AuditLogType(auditLogModel.Type),
		Content:   content,
		CreatedAt: auditLogModel.CreatedAt,
	}, nil
}

// Create 새 감사 로그 생성
func (r *AuditLogRepositoryImpl) Create(ctx context.Context, log *entity.AuditLog) error {
	auditLogModel, err := toAuditLogModel(log)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Create(auditLogModel).Error; err != nil {
		return fmt.Errorf("감사 로그 생성 실패: %w", err)
	}

	log.ID = auditLogModel.ID
	return nil
}

// GetByID ID로 감사 로그 조회
func (r *AuditLogRepositoryImpl) GetByID(ctx context.Context, id uint) (*entity.AuditLog, error) {
	var auditLogModel model.AuditLogModel

	if err := r.db.WithContext(ctx).First(&auditLogModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 감사 로그를 찾지 못함
		}
		return nil, fmt.Errorf("감사 로그 조회 실패: %w", err)
	}

	return toAuditLogEntity(&auditLogModel)
}

// ListByUserID 사용자 ID로 감사 로그 목록 조회
func (r *AuditLogRepositoryImpl) ListByUserID(ctx context.Context, userID string, page, limit int) ([]*entity.AuditLog, int64, error) {
	var auditLogModels []model.AuditLogModel
	var total int64

	// 전체 개수 카운트
	if err := r.db.WithContext(ctx).Model(&model.AuditLogModel{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("감사 로그 개수 조회 실패: %w", err)
	}

	// 페이징 처리된 데이터 조회
	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&auditLogModels).Error; err != nil {
		return nil, 0, fmt.Errorf("감사 로그 목록 조회 실패: %w", err)
	}

	// 도메인 엔티티로 변환
	auditLogs := make([]*entity.AuditLog, len(auditLogModels))
	for i, m := range auditLogModels {
		auditLog, err := toAuditLogEntity(&m)
		if err != nil {
			return nil, 0, err
		}
		auditLogs[i] = auditLog
	}

	return auditLogs, total, nil
}

// ListByType 로그 유형으로 감사 로그 목록 조회
func (r *AuditLogRepositoryImpl) ListByType(ctx context.Context, logType entity.AuditLogType, page, limit int) ([]*entity.AuditLog, int64, error) {
	var auditLogModels []model.AuditLogModel
	var total int64

	// 전체 개수 카운트
	if err := r.db.WithContext(ctx).Model(&model.AuditLogModel{}).Where("type = ?", string(logType)).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("감사 로그 개수 조회 실패: %w", err)
	}

	// 페이징 처리된 데이터 조회
	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Where("type = ?", string(logType)).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&auditLogModels).Error; err != nil {
		return nil, 0, fmt.Errorf("감사 로그 목록 조회 실패: %w", err)
	}

	// 도메인 엔티티로 변환
	auditLogs := make([]*entity.AuditLog, len(auditLogModels))
	for i, m := range auditLogModels {
		auditLog, err := toAuditLogEntity(&m)
		if err != nil {
			return nil, 0, err
		}
		auditLogs[i] = auditLog
	}

	return auditLogs, total, nil
}

// Search 검색 조건으로 감사 로그 조회
func (r *AuditLogRepositoryImpl) Search(
	ctx context.Context,
	userID *string,
	logTypes []entity.AuditLogType,
	startDate, endDate *string,
	page, limit int,
) ([]*entity.AuditLog, int64, error) {
	var auditLogModels []model.AuditLogModel
	var total int64

	db := r.db.WithContext(ctx).Model(&model.AuditLogModel{})

	// 검색 조건 적용
	if userID != nil && *userID != "" {
		db = db.Where("user_id = ?", *userID)
	}

	if len(logTypes) > 0 {
		typeStrings := make([]string, len(logTypes))
		for i, t := range logTypes {
			typeStrings[i] = string(t)
		}
		db = db.Where("type IN ?", typeStrings)
	}

	if startDate != nil && *startDate != "" {
		parsedStartDate, err := time.Parse("2006-01-02", *startDate)
		if err == nil {
			db = db.Where("created_at >= ?", parsedStartDate)
		}
	}

	if endDate != nil && *endDate != "" {
		parsedEndDate, err := time.Parse("2006-01-02", *endDate)
		if err == nil {
			// 하루를 더해서 날짜 범위를 포함시킴
			parsedEndDate = parsedEndDate.Add(24 * time.Hour)
			db = db.Where("created_at < ?", parsedEndDate)
		}
	}

	// 전체 개수 카운트
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("감사 로그 개수 조회 실패: %w", err)
	}

	// 페이징 처리된 데이터 조회
	offset := (page - 1) * limit
	if err := db.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&auditLogModels).Error; err != nil {
		return nil, 0, fmt.Errorf("감사 로그 목록 조회 실패: %w", err)
	}

	// 도메인 엔티티로 변환
	auditLogs := make([]*entity.AuditLog, len(auditLogModels))
	for i, m := range auditLogModels {
		auditLog, err := toAuditLogEntity(&m)
		if err != nil {
			return nil, 0, err
		}
		auditLogs[i] = auditLog
	}

	return auditLogs, total, nil
}

// Delete 감사 로그 삭제
func (r *AuditLogRepositoryImpl) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.AuditLogModel{}, id).Error; err != nil {
		return fmt.Errorf("감사 로그 삭제 실패: %w", err)
	}
	return nil
}
