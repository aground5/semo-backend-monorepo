package repository

import (
	"context"
	"time"
)

// ActivityType은 활동 유형을 정의합니다
type ActivityType string

const (
	ActivityTypeCreate  ActivityType = "create"
	ActivityTypeUpdate  ActivityType = "update"
	ActivityTypeDelete  ActivityType = "delete"
	ActivityTypeShare   ActivityType = "share"
	ActivityTypeJoin    ActivityType = "join"
	ActivityTypeLeave   ActivityType = "leave"
	ActivityTypeComment ActivityType = "comment"
)

// ActivityInfo는 활동 정보를 표현합니다
type ActivityInfo struct {
	ID          string
	Type        ActivityType
	ItemID      string
	ProfileID   string
	Description string
	Data        map[string]interface{}
	CreatedAt   time.Time
}

// ActivityRepository 활동 관련 저장소 인터페이스
type ActivityRepository interface {
	// Create 새 활동 정보 생성
	Create(ctx context.Context, activity *ActivityInfo) error

	// FindByID ID로 활동 정보 조회
	FindByID(ctx context.Context, id string) (*ActivityInfo, error)

	// FindByItemID 아이템 ID로 활동 정보 목록 조회
	FindByItemID(ctx context.Context, itemID string, limit, offset int) ([]*ActivityInfo, error)

	// FindByProfileID 프로필 ID로 활동 정보 목록 조회
	FindByProfileID(ctx context.Context, profileID string, limit, offset int) ([]*ActivityInfo, error)

	// FindByType 활동 유형으로 활동 정보 목록 조회
	FindByType(ctx context.Context, activityType ActivityType, limit, offset int) ([]*ActivityInfo, error)

	// FindRecent 최근 활동 정보 목록 조회
	FindRecent(ctx context.Context, limit, offset int) ([]*ActivityInfo, error)

	// CountByItemID 아이템 ID로 활동 개수 조회
	CountByItemID(ctx context.Context, itemID string) (int64, error)
}
