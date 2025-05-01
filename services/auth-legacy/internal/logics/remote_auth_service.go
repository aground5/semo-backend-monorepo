package logics

import (
	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RemoteAuthService provides remote session management functions
type RemoteAuthService struct{}

// NewRemoteAuthService creates a new RemoteAuthService
func NewRemoteAuthService() *RemoteAuthService {
	return &RemoteAuthService{}
}

// GetActiveActivities returns all active sessions for a user
func (s *RemoteAuthService) GetActiveActivities(userID string) ([]models.Activity, error) {
	var activities []models.Activity
	err := repositories.DBS.Postgres.
		Where("logout_at IS NULL AND user_id = ?", userID).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// DeactivateActivity forcibly logs out a session by its ID
// It deletes the session in Redis, revokes the token group,
// and marks the activity as logged out
func (s *RemoteAuthService) DeactivateActivity(sessionID string, userID string) error {
	// 1. Find the activity (that hasn't been logged out yet)
	var activity models.Activity
	err := repositories.DBS.Postgres.
		Where("session_id = ? AND user_id = ? AND logout_at IS NULL", sessionID, userID).
		First(&activity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("activity not found or already deactivated")
		}
		return err
	}

	// 2. Delete the session from Redis
	redisKey := "session:" + activity.SessionID
	ctx := context.Background()
	if err := repositories.DBS.Redis.Del(ctx, redisKey).Err(); err != nil {
		configs.Logger.Error("failed to delete session from redis", zap.Error(err))
		// Continue with forced logout even if Redis deletion fails
	}

	// 3. Delete the token group (if it exists)
	var tokenGroup models.TokenGroup
	tokenGroup.ID = activity.TokenGroupID
	if activity.TokenGroupID != 0 {
		if err := repositories.DBS.Postgres.Delete(&tokenGroup).Error; err != nil {
			configs.Logger.Error("failed to delete token group", zap.Error(err))
			return err
		}
	}

	// 4. Update activity with logout timestamp
	now := time.Now()
	if err := repositories.DBS.Postgres.
		Model(&activity).
		Update("logout_at", now).Error; err != nil {
		configs.Logger.Error("failed to update activity logout time", zap.Error(err))
		return err
	}

	configs.Logger.Info("activity deactivated",
		zap.String("sessionID", activity.SessionID),
		zap.String("userID", activity.UserID),
	)
	return nil
}

// Global instance of RemoteAuthService
var RemoteAuthSvc = NewRemoteAuthService()
