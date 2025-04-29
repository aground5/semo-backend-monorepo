package logics

import (
	"fmt"

	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"
)

type ShareService struct {
	cursorManager *utils.CursorManager
}

func NewShareService(cursorManager *utils.CursorManager) *ShareService {
	return &ShareService{
		cursorManager: cursorManager,
	}
}

type ShareResult struct {
	Shares []models.Share `json:"shares"`
	utils.PaginationResult
}

// GetShareUUID 태스크 ID와 프로필 ID를 받아 UUID를 반환하는 메서드
func (ss *ShareService) GetShareUUID(rootTaskID, profileID string) (string, error) {
	var share models.Share
	if err := repositories.DBS.Postgres.First(&share, "root_task_id = ? AND created_by = ?", rootTaskID, profileID).Error; err != nil {
		return "", fmt.Errorf("share not found: %w", err)
	}
	return share.ID, nil
}

// CreateShare creates a new share in the database
func (ss *ShareService) CreateShare(share *models.Share) (*models.Share, error) {
	// Create share in database
	if err := repositories.DBS.Postgres.Create(share).Error; err != nil {
		return nil, fmt.Errorf("failed to create share: %w", err)
	}

	// Reload the share to get all relations
	var result models.Share
	if err := repositories.DBS.Postgres.
		Preload("RootTask").
		First(&result, "id = ?", share.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load created share: %w", err)
	}

	return &result, nil
}

// DeleteShare 공유를 삭제하는 메서드
func (ss *ShareService) DeleteShare(uuid string) error {
	var share models.Share
	if err := repositories.DBS.Postgres.First(&share, "id = ?", uuid).Error; err != nil {
		return fmt.Errorf("share not found: %w", err)
	}

	if err := repositories.DBS.Postgres.Delete(&share).Error; err != nil {
		return fmt.Errorf("failed to revoke share: %w", err)
	}

	return nil
}

// CheckShareExists checks if a share exists for the given rootTaskID and profileID
func (ss *ShareService) CheckShareExists(rootTaskID, profileID string) (bool, error) {
	var count int64
	if err := repositories.DBS.Postgres.Model(&models.Share{}).
		Where("root_task_id = ? AND created_by = ?", rootTaskID, profileID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check share existence: %w", err)
	}

	return count > 0, nil
}
