package repository

import (
	"context"
	"time"
)

// ShareInfo는 공유 정보를 표현합니다
type ShareInfo struct {
	ID          string
	ItemID      string
	SharedBy    string
	SharedTo    string
	SharedEmail string
	Role        string
	Token       string
	IsAccepted  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   *time.Time
}

// ShareRepository 공유 관련 저장소 인터페이스
type ShareRepository interface {
	// FindByID ID로 공유 정보 조회
	FindByID(ctx context.Context, id string) (*ShareInfo, error)

	// FindByItemID 아이템 ID로 공유 정보 목록 조회
	FindByItemID(ctx context.Context, itemID string) ([]*ShareInfo, error)

	// FindBySharedBy 공유자로 공유 정보 목록 조회
	FindBySharedBy(ctx context.Context, sharedBy string) ([]*ShareInfo, error)

	// FindBySharedTo 공유 대상자로 공유 정보 목록 조회
	FindBySharedTo(ctx context.Context, sharedTo string) ([]*ShareInfo, error)

	// FindBySharedEmail 공유 이메일로 공유 정보 목록 조회
	FindBySharedEmail(ctx context.Context, email string) ([]*ShareInfo, error)

	// Create 새 공유 정보 생성
	Create(ctx context.Context, share *ShareInfo) error

	// Update 공유 정보 업데이트
	Update(ctx context.Context, share *ShareInfo) error

	// Delete 공유 정보 삭제
	Delete(ctx context.Context, id string) error

	// AcceptShare 공유 수락
	AcceptShare(ctx context.Context, id string) error

	// VerifyToken 공유 토큰 검증
	VerifyToken(ctx context.Context, token string) (*ShareInfo, error)

	// FindByToken 토큰으로 공유 정보 조회
	FindByToken(ctx context.Context, token string) (*ShareInfo, error)
}
