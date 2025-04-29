package collectors

import (
	"errors"
	"fmt"

	"semo-server/internal/models"
	"semo-server/internal/repositories"
)

// TaskCollector collects task data
type TaskCollector struct{}

// NewTaskCollector creates a new TaskCollector
func NewTaskCollector() *TaskCollector {
	return &TaskCollector{}
}

// GetTask retrieves a task by ID
func (c *TaskCollector) GetTask(taskID string) (*models.Item, error) {
	var task models.Item
	if err := repositories.DBS.Postgres.Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &task, nil
}

// GetTaskWithChildren retrieves a task with its children
func (c *TaskCollector) GetTaskWithChildren(taskID string, maxLevel int) (*models.Item, []models.Item, error) {
	// Get the main task
	task, err := c.GetTask(taskID)
	if err != nil {
		return nil, nil, err
	}

	// Get all children up to the specified level
	var children []models.Item
	if maxLevel > 0 {
		// Implement a recursive query or multiple queries to get children up to maxLevel
		// For simplicity, this example just gets the immediate children
		if err := repositories.DBS.Postgres.Where("parent_id = ? AND type = ?", taskID, "task").Order("ordering ASC").Find(&children).Error; err != nil {
			return task, nil, fmt.Errorf("failed to get children: %w", err)
		}
	}

	return task, children, nil
}

// GetTaskSubtree gets all tasks in the subtree
func (c *TaskCollector) GetTaskSubtree(taskID string, maxLevel int) ([]models.Item, error) {
	if taskID == "" {
		return nil, errors.New("task ID is empty")
	}

	// Get the root task
	root, err := c.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	// Initialize result with the root task
	result := []models.Item{*root}

	// If maxLevel is 0, only return the root
	if maxLevel == 0 {
		return result, nil
	}

	// Get all children recursively
	// This is a simplified implementation; in practice, you might want to use a recursive query
	var getChildren func(parentID string, currentLevel int) ([]models.Item, error)
	getChildren = func(parentID string, currentLevel int) ([]models.Item, error) {
		if currentLevel >= maxLevel {
			return nil, nil
		}

		var children []models.Item
		if err := repositories.DBS.Postgres.Where("parent_id = ? AND type = ?", parentID, "task").Order("position ASC").Find(&children).Error; err != nil {
			return nil, fmt.Errorf("failed to get children: %w", err)
		}

		result := make([]models.Item, 0, len(children))
		for _, child := range children {
			result = append(result, child)

			// Get grandchildren
			grandchildren, err := getChildren(child.ID, currentLevel+1)
			if err != nil {
				return nil, err
			}
			result = append(result, grandchildren...)
		}

		return result, nil
	}

	// Get all children
	children, err := getChildren(taskID, 1)
	if err != nil {
		return nil, err
	}

	// Add children to result
	result = append(result, children...)

	return result, nil
}
