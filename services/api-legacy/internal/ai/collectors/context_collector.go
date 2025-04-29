package collectors

import (
	"fmt"

	"semo-server/internal/ai/formatters"
)

// ContextCollector collects context data for AI requests
type ContextCollector struct {
	TaskCollector     *TaskCollector
	UserDataCollector *UserDataCollector
	TaskFormatter     *formatters.TaskFormatter
}

// NewContextCollector creates a new ContextCollector
func NewContextCollector() *ContextCollector {
	taskCollector := NewTaskCollector()
	return &ContextCollector{
		TaskCollector:     taskCollector,
		UserDataCollector: NewUserDataCollector(),
		TaskFormatter:     formatters.NewTaskFormatter(taskCollector),
	}
}

// CollectContextForTask collects context data for a task
func (c *ContextCollector) CollectContextForTask(taskID string, includeUserData bool, userID string) (map[string]any, error) {
	// Get task
	task, err := c.TaskCollector.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Get formatted task
	formattedTask, err := c.TaskFormatter.GetFormattedTaskContext(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get formatted task: %w", err)
	}

	// Prepare context data
	contextData := map[string]any{
		"TodoList":     formattedTask,
		"Language":     "KOREAN",
		"SelectedTodo": task.Name,
		"UserData":     "",
		"Reference":    "",
	}

	// Add user data if requested
	if includeUserData {
		userData, err := c.UserDataCollector.GetCombinedUserData(taskID, userID)
		if err == nil && userData != "" {
			contextData["UserData"] = userData
		}
	}

	return contextData, nil
}
