package logics

import (
	"errors"
	"fmt"
	"semo-server/configs"
	"semo-server/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserTestService provides operations for managing user tests
type UserTestService struct {
	db *gorm.DB
}

// NewUserTestService creates a new instance of UserTestService
func NewUserTestService(db *gorm.DB) *UserTestService {
	return &UserTestService{db: db}
}

// CreateUserTest creates a new UserTests record with the given taskID and question
// fulfills requirement #1: taskID에 맞는 question을 담은 UserTests 생성하기
func (s *UserTestService) CreateUserTest(taskID string, question string, userID string) (*models.UserTests, error) {
	if taskID == "" {
		return nil, errors.New("taskID is required")
	}
	if question == "" {
		return nil, errors.New("question is required")
	}
	if userID == "" {
		return nil, errors.New("userID is required")
	}

	userTest := &models.UserTests{
		TaskID:   taskID,
		Question: question,
		Answer:   "", // Initialize with empty answer
		UserData: "", // Initialize with empty user data
		UserID:   userID,
	}

	result := s.db.Create(userTest)
	if result.Error != nil {
		configs.Logger.Error("Failed to create user test",
			zap.String("taskID", taskID),
			zap.Error(result.Error))
		return nil, fmt.Errorf("failed to create user test: %w", result.Error)
	}

	configs.Logger.Info("Created user test",
		zap.String("taskID", taskID),
		zap.Int("id", userTest.ID))
	return userTest, nil
}

// UpdateLatestAnswer updates the answer field of the most recent UserTests for a taskID
// fulfills requirement #2: taskID에 맞는 UserTests 중 제일 ID 값이 큰 record의 answer 값을 업데이트 하는 함수
func (s *UserTestService) UpdateLatestAnswer(taskID string, answer string, userID string) (*models.UserTests, error) {
	if taskID == "" {
		return nil, errors.New("taskID is required")
	}
	if answer == "" {
		return nil, errors.New("answer is required")
	}
	if userID == "" {
		return nil, errors.New("userID is required")
	}

	// Find the most recent UserTests record for the taskID (the one with highest ID)
	var userTest models.UserTests
	result := s.db.Where("task_id = ? AND user_id = ?", taskID, userID).
		Order("id DESC").
		First(&userTest)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			configs.Logger.Warn("No UserTests found for taskID and userID",
				zap.String("taskID", taskID),
				zap.String("userID", userID))
			return nil, fmt.Errorf("no UserTests found for taskID %s and userID %s", taskID, userID)
		}
		configs.Logger.Error("Failed to fetch latest UserTests",
			zap.String("taskID", taskID),
			zap.String("userID", userID),
			zap.Error(result.Error))
		return nil, fmt.Errorf("failed to fetch latest UserTests: %w", result.Error)
	}

	// Update the answer
	userTest.Answer = answer
	updateResult := s.db.Save(&userTest)
	if updateResult.Error != nil {
		configs.Logger.Error("Failed to update answer",
			zap.Int("id", userTest.ID),
			zap.Error(updateResult.Error))
		return nil, fmt.Errorf("failed to update answer: %w", updateResult.Error)
	}

	configs.Logger.Info("Updated answer for user test",
		zap.Int("id", userTest.ID),
		zap.String("taskID", taskID))
	return &userTest, nil
}

// UpdateUserData updates the UserData field of a UserTests record by ID
// fulfills requirement #3: ID 에 맞게 USERDATA 값을 업데이트 하는 함수
func (s *UserTestService) UpdateUserData(id int, userData string, userID string) (*models.UserTests, error) {
	if id <= 0 {
		return nil, errors.New("valid ID is required")
	}
	if userID == "" {
		return nil, errors.New("userID is required")
	}

	// Find the UserTests by ID and userID
	var userTest models.UserTests
	result := s.db.Where("id = ? AND user_id = ?", id, userID).First(&userTest)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			configs.Logger.Warn("UserTests not found for this user",
				zap.Int("id", id),
				zap.String("userID", userID))
			return nil, fmt.Errorf("UserTests with ID %d not found for this user", id)
		}
		configs.Logger.Error("Failed to fetch UserTests",
			zap.Int("id", id),
			zap.Error(result.Error))
		return nil, fmt.Errorf("failed to fetch UserTests: %w", result.Error)
	}

	// Update the UserData
	userTest.UserData = userData
	updateResult := s.db.Save(&userTest)
	if updateResult.Error != nil {
		configs.Logger.Error("Failed to update UserData",
			zap.Int("id", id),
			zap.Error(updateResult.Error))
		return nil, fmt.Errorf("failed to update UserData: %w", updateResult.Error)
	}

	configs.Logger.Info("Updated UserData for user test", zap.Int("id", id))
	return &userTest, nil
}
