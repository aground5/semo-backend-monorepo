package repository

import (
	"context"
	"time"
)

// CommentInfo는 댓글 정보를 표현합니다
type CommentInfo struct {
	ID        string
	ItemID    string
	ParentID  *string
	Content   string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CommentRepository 댓글 관련 저장소 인터페이스
type CommentRepository interface {
	// FindByID ID로 댓글 정보 조회
	FindByID(ctx context.Context, id string) (*CommentInfo, error)

	// FindByItemID 아이템 ID로 댓글 목록 조회
	FindByItemID(ctx context.Context, itemID string, limit, offset int) ([]*CommentInfo, error)

	// FindReplies 부모 댓글에 대한 답글 목록 조회
	FindReplies(ctx context.Context, parentID string, limit, offset int) ([]*CommentInfo, error)

	// Create 새 댓글 생성
	Create(ctx context.Context, comment *CommentInfo) error

	// Update 댓글 정보 업데이트
	Update(ctx context.Context, comment *CommentInfo) error

	// Delete 댓글 삭제
	Delete(ctx context.Context, id string) error

	// CountByItemID 아이템 ID로 댓글 개수 조회
	CountByItemID(ctx context.Context, itemID string) (int64, error)

	// CountReplies 부모 댓글에 대한 답글 개수 조회
	CountReplies(ctx context.Context, parentID string) (int64, error)
}
