package collectors

import (
	"errors"
	"fmt"
	"semo-server/configs-legacy"

	"go.uber.org/zap"

	"semo-server/internal/ai/formatters"
	"semo-server/internal/models"
	"semo-server/internal/repositories"
)

// UserDataCollector collects user data
type UserDataCollector struct {
	formatter *formatters.UserDataFormatter
}

// NewUserDataCollector creates a new UserDataCollector
func NewUserDataCollector() *UserDataCollector {
	return &UserDataCollector{
		formatter: formatters.NewUserDataFormatter(),
	}
}

// GetUserTest retrieves a user test by ID, filtered by userID
func (c *UserDataCollector) GetUserTest(id int, userID string) (*models.UserTests, error) {
	var userTest models.UserTests
	if err := repositories.DBS.Postgres.Where("id = ? AND user_id = ?", id, userID).First(&userTest).Error; err != nil {
		return nil, fmt.Errorf("failed to get user test: %w", err)
	}
	return &userTest, nil
}

// GetLatestUserTest retrieves the latest user test for a task, filtered by userID
func (c *UserDataCollector) GetLatestUserTest(taskID string, userID string) (*models.UserTests, error) {
	var userTest models.UserTests
	if err := repositories.DBS.Postgres.Where("task_id = ? AND user_id = ?", taskID, userID).Order("created_at DESC").First(&userTest).Error; err != nil {
		return nil, fmt.Errorf("failed to get latest user test: %w", err)
	}
	return &userTest, nil
}

// GetAllUserTests retrieves all user tests for a task, filtered by userID
func (c *UserDataCollector) GetAllUserTests(taskID string, userID string) ([]models.UserTests, error) {
	var userTests []models.UserTests
	if err := repositories.DBS.Postgres.Where("task_id = ? AND user_id = ?", taskID, userID).Order("created_at DESC").Find(&userTests).Error; err != nil {
		return nil, fmt.Errorf("failed to get user tests: %w", err)
	}
	return userTests, nil
}

// validateInputs checks if the required inputs are provided
func (c *UserDataCollector) validateInputs(taskID string, userID string) error {
	if taskID == "" {
		return errors.New("taskID is required")
	}
	if userID == "" {
		return errors.New("userID is required")
	}
	return nil
}

// GetTaskByID retrieves a task by its ID
func (c *UserDataCollector) GetTaskByID(taskID string) (*models.Item, error) {
	var task models.Item
	if err := repositories.DBS.Postgres.First(&task, "id = ?", taskID).Error; err != nil {
		configs.Logger.Error("Failed to get task",
			zap.String("taskID", taskID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get task with ID %s: %w", taskID, err)
	}
	return &task, nil
}

// GetTaskChain retrieves the chain of tasks from root to the specified task
func (c *UserDataCollector) GetTaskChain(taskID string) ([]models.Item, error) {
	// Get the initial task
	task, err := c.GetTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	// Build the chain of tasks starting with the current one
	taskChain := []models.Item{*task}

	// Traverse up the parent chain until we reach a task with no parent (the root)
	currentTask := *task
	for currentTask.ParentID != nil && *currentTask.ParentID != "" {
		// Get the parent task
		parentTask, err := c.GetTaskByID(*currentTask.ParentID)
		if err != nil {
			return nil, err
		}

		// Insert at the beginning to maintain parent->child order
		taskChain = append([]models.Item{*parentTask}, taskChain...)

		// Move up to the next parent
		currentTask = *parentTask
	}

	configs.Logger.Info("Built task chain",
		zap.String("taskID", taskID),
		zap.Int("chainLength", len(taskChain)))

	return taskChain, nil
}

// GetCombinedUserData retrieves all user test data for a task and its parents
// Returns combined userData string in parent -> child order
func (c *UserDataCollector) GetCombinedUserData(taskID string, userID string) (string, error) {
	// Validate inputs
	if err := c.validateInputs(taskID, userID); err != nil {
		return "", err
	}

	// Get task chain (parent -> child order)
	taskChain, err := c.GetTaskChain(taskID)
	if err != nil {
		return "", err
	}

	// Collect user tests for each task in the chain
	var userTestsByTask [][]models.UserTests

	// Process each task in order (parent -> child)
	for _, task := range taskChain {
		// Get user tests for this task
		userTests, err := c.GetAllUserTests(task.ID, userID)
		if err != nil {
			// Skip this task but continue with others
			continue
		}

		// Only add if there are tests
		if len(userTests) > 0 {
			userTestsByTask = append(userTestsByTask, userTests)
		}
	}

	// Format the combined data using the formatter
	combinedData := c.formatter.FormatCombinedUserData(userTestsByTask)

	configs.Logger.Info("Generated combined user data",
		zap.String("taskID", taskID),
		zap.Int("parentCount", len(taskChain)-1))

	return combinedData, nil
}
