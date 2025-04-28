package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db/model"
	"gorm.io/gorm"
)

type ActivityRepositoryImpl struct {
	db *gorm.DB
}

// NewActivityRepository 활동 저장소 구현체 생성
func NewActivityRepository(db *gorm.DB) repository.ActivityRepository {
	return &ActivityRepositoryImpl{db: db}
}

// 도메인 엔티티를 DB 모델로 변환
func toActivityModel(activity *entity.Activity) *model.ActivityModel {
	var deviceUID *uuid.UUID
	if activity.DeviceUID != "" {
		uid, err := uuid.Parse(activity.DeviceUID)
		if err == nil {
			deviceUID = &uid
		}
	}

	return &model.ActivityModel{
		SessionID:    activity.SessionID,
		UserID:       activity.UserID,
		TokenGroupID: activity.TokenGroupID,
		IP:           activity.IP,
		UserAgent:    activity.UserAgent,
		DeviceUID:    deviceUID,
		LoginAt:      activity.LoginAt,
		LogoutAt:     activity.LogoutAt,
		LocationInfo: activity.LocationInfo,
		DeviceInfo:   activity.DeviceInfo,
	}
}

// DB 모델을 도메인 엔티티로 변환
func toActivityEntity(model *model.ActivityModel) *entity.Activity {
	deviceUID := ""
	if model.DeviceUID != nil {
		deviceUID = model.DeviceUID.String()
	}

	return &entity.Activity{
		SessionID:    model.SessionID,
		UserID:       model.UserID,
		TokenGroupID: model.TokenGroupID,
		IP:           model.IP,
		UserAgent:    model.UserAgent,
		DeviceUID:    deviceUID,
		LoginAt:      model.LoginAt,
		LogoutAt:     model.LogoutAt,
		LocationInfo: model.LocationInfo,
		DeviceInfo:   model.DeviceInfo,
	}
}

// FindBySessionID 세션 ID로 활동 기록 조회
func (r *ActivityRepositoryImpl) FindBySessionID(ctx context.Context, sessionID string) (*entity.Activity, error) {
	var activityModel model.ActivityModel

	if err := r.db.WithContext(ctx).First(&activityModel, "session_id = ?", sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 활동 기록을 찾지 못함
		}
		return nil, err
	}

	return toActivityEntity(&activityModel), nil
}

// FindByUserID 사용자 ID로 활동 기록 조회
func (r *ActivityRepositoryImpl) FindByUserID(ctx context.Context, userID string) ([]*entity.Activity, error) {
	var activityModels []model.ActivityModel

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&activityModels).Error; err != nil {
		return nil, err
	}

	activities := make([]*entity.Activity, len(activityModels))
	for i, model := range activityModels {
		activities[i] = toActivityEntity(&model)
	}

	return activities, nil
}

// FindActiveSessions 사용자의 활성 세션 목록 조회
func (r *ActivityRepositoryImpl) FindActiveSessions(ctx context.Context, userID string) ([]*entity.Activity, error) {
	var activityModels []model.ActivityModel

	if err := r.db.WithContext(ctx).Where("user_id = ? AND logout_at IS NULL", userID).Find(&activityModels).Error; err != nil {
		return nil, err
	}

	activities := make([]*entity.Activity, len(activityModels))
	for i, model := range activityModels {
		activities[i] = toActivityEntity(&model)
	}

	return activities, nil
}

// Create 새 활동 기록 생성
func (r *ActivityRepositoryImpl) Create(ctx context.Context, activity *entity.Activity) error {
	activityModel := toActivityModel(activity)

	return r.db.WithContext(ctx).Create(activityModel).Error
}

// Update 활동 기록 업데이트
func (r *ActivityRepositoryImpl) Update(ctx context.Context, activity *entity.Activity) error {
	activityModel := toActivityModel(activity)

	return r.db.WithContext(ctx).Save(activityModel).Error
}

// Delete 활동 기록 삭제
func (r *ActivityRepositoryImpl) Delete(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Delete(&model.ActivityModel{}, "session_id = ?", sessionID).Error
}
