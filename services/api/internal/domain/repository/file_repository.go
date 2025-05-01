package repository

import (
	"context"
	"io"
	"time"
)

// FileInfo는 파일 정보를 표현합니다
type FileInfo struct {
	ID        string
	ItemID    string
	Name      string
	Path      string
	Size      int64
	Type      string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FileRepository 파일 관련 저장소 인터페이스
type FileRepository interface {
	// FindByID ID로 파일 정보 조회
	FindByID(ctx context.Context, id string) (*FileInfo, error)

	// FindByItemID 아이템 ID로 파일 목록 조회
	FindByItemID(ctx context.Context, itemID string) ([]*FileInfo, error)

	// Create 새 파일 저장
	Create(ctx context.Context, file *FileInfo, reader io.Reader) error

	// Delete 파일 삭제
	Delete(ctx context.Context, id string) error

	// GetDownloadURL 다운로드 URL 생성
	GetDownloadURL(ctx context.Context, id string) (string, error)

	// Update 파일 정보 업데이트
	Update(ctx context.Context, file *FileInfo) error

	// GetReader 파일 내용을 읽기 위한 Reader 반환
	GetReader(ctx context.Context, id string) (io.ReadCloser, error)
}
