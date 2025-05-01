package repository

import (
	"context"
	"time"
)

// ProfileInfo는 사용자 프로필 정보를 표현합니다
type ProfileInfo struct {
	ID          string
	UserID      string
	Username    string
	Email       string
	Name        string
	ImageURL    string
	PhoneNumber string
	Bio         string
	Job         string
	Setting     map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProfileRepository 프로필 관련 저장소 인터페이스
type ProfileRepository interface {
	// FindByID ID로 프로필 정보 조회
	FindByID(ctx context.Context, id string) (*ProfileInfo, error)

	// FindByUserID 사용자 ID로 프로필 정보 조회
	FindByUserID(ctx context.Context, userID string) (*ProfileInfo, error)

	// FindByEmail 이메일로 프로필 정보 조회
	FindByEmail(ctx context.Context, email string) (*ProfileInfo, error)

	// FindByUsername 사용자명으로 프로필 정보 조회
	FindByUsername(ctx context.Context, username string) (*ProfileInfo, error)

	// Create 새 프로필 생성
	Create(ctx context.Context, profile *ProfileInfo) error

	// Update 프로필 정보 업데이트
	Update(ctx context.Context, profile *ProfileInfo) error

	// Delete 프로필 삭제
	Delete(ctx context.Context, id string) error

	// UpdateSetting 프로필 설정 업데이트
	UpdateSetting(ctx context.Context, id string, setting map[string]interface{}) error

	// FindByTeam 팀에 속한 프로필 목록 조회
	FindByTeam(ctx context.Context, teamID string) ([]*ProfileInfo, error)

	// Search 프로필 검색
	Search(ctx context.Context, query string, limit, offset int) ([]*ProfileInfo, error)
}
