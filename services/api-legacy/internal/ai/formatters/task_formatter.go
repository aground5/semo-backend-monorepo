package formatters

import (
	"fmt"
	"strings"

	"semo-server/internal/models"
	"semo-server/internal/repositories"
)

// TaskFormatter formats tasks for AI context
type TaskFormatter struct {
	taskCollector TaskCollectorInterface
}

// TaskCollectorInterface defines the interface for task collection
type TaskCollectorInterface interface {
	GetTask(taskID string) (*models.Item, error)
}

// NewTaskFormatter creates a new TaskFormatter
func NewTaskFormatter(taskCollector TaskCollectorInterface) *TaskFormatter {
	return &TaskFormatter{
		taskCollector: taskCollector,
	}
}

// GetFormattedTaskContext formats a task and its context for use in prompts
// It creates a hierarchical representation of tasks following the path to the target task
func (f *TaskFormatter) GetFormattedTaskContext(taskID string) (string, error) {
	// Get the task
	task, err := f.taskCollector.GetTask(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get task with ID %s: %w", taskID, err)
	}

	// Check if the task is of type "task"
	if task.Type != "task" {
		return "", fmt.Errorf("task with ID %s is not of type 'task'", taskID)
	}

	// Find the path from task to root
	pathToRoot, err := f.buildPathToRoot(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to build path to root: %w", err)
	}

	// Create a map of path node IDs for quick lookup
	pathNodeIDs := make(map[string]bool)
	for _, node := range pathToRoot {
		pathNodeIDs[node.ID] = true
	}

	// Reverse the path to get root to task order
	var pathFromRoot []models.Item
	for i := len(pathToRoot) - 1; i >= 0; i-- {
		pathFromRoot = append(pathFromRoot, pathToRoot[i])
	}

	// Start building the formatted output
	var sb strings.Builder

	// Start with the root node as a heading
	rootTask := pathFromRoot[0]
	sb.WriteString(fmt.Sprintf("# %s\n", rootTask.Name))

	// Get all first level children (direct children of root)
	var rootChildren []models.Item
	if err := repositories.DBS.Postgres.Where("parent_id = ?", rootTask.ID).Order("position ASC").Find(&rootChildren).Error; err != nil {
		return "", fmt.Errorf("failed to get root children: %w", err)
	}

	// Process first level children and continue with hierarchy
	for i, child := range rootChildren {
		// Add first level numbering
		prefix := fmt.Sprintf("%d.", i+1)
		sb.WriteString(fmt.Sprintf("%s %s\n", prefix, child.Name))

		// Add contents if available
		if child.Objective != "" && strings.TrimSpace(child.Objective) != "" {
			sb.WriteString("   - Objective: " + strings.TrimSpace(child.Objective) + "\n")
		}
		if child.Deliverable != "" && strings.TrimSpace(child.Deliverable) != "" {
			sb.WriteString("   - Deliverable: " + strings.TrimSpace(child.Deliverable) + "\n")
		}

		// If this child is in our path, process its children as well
		if pathNodeIDs[child.ID] {
			// Process children hierarchically with increasing prefix
			if err := f.appendChildrenHierarchy(&sb, child.ID, prefix, pathNodeIDs); err != nil {
				return "", fmt.Errorf("failed to append children hierarchy: %w", err)
			}
		}
	}

	return strings.TrimSpace(sb.String()), nil
}

// buildPathToRoot builds a path from the task to the root
func (f *TaskFormatter) buildPathToRoot(taskID string) ([]models.Item, error) {
	var path []models.Item

	// Get the task
	task, err := f.taskCollector.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task with ID %s: %w", taskID, err)
	}

	// Add the current task to the path
	currentTask := task
	path = append(path, *currentTask)

	// Traverse up the parent hierarchy until we reach the root
	for currentTask.ParentID != nil && *currentTask.ParentID != "" {
		parentTask, err := f.taskCollector.GetTask(*currentTask.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent task with ID %s: %w", *currentTask.ParentID, err)
		}

		// Add the parent to the path
		path = append(path, *parentTask)

		// Move to the parent
		currentTask = parentTask
	}

	return path, nil
}

// appendChildrenHierarchy recursively appends children to the string builder with proper numbering
func (f *TaskFormatter) appendChildrenHierarchy(sb *strings.Builder, parentID string, parentPrefix string, pathNodeIDs map[string]bool) error {
	// Get all children of this parent
	var children []models.Item
	if err := repositories.DBS.Postgres.Where("parent_id = ?", parentID).Order("position ASC").Find(&children).Error; err != nil {
		return fmt.Errorf("failed to get children of parent %s: %w", parentID, err)
	}

	// Process each child
	for i, child := range children {
		// Create child prefix based on parent prefix (e.g., "1.2.")
		childPrefix := fmt.Sprintf("%s%d.", parentPrefix, i+1)

		// Add child to output
		sb.WriteString(fmt.Sprintf("%s %s\n", childPrefix, child.Name))

		// Add contents if available
		if child.Objective != "" && strings.TrimSpace(child.Objective) != "" {
			sb.WriteString("   - Objective: " + strings.TrimSpace(child.Objective) + "\n")
		}
		if child.Deliverable != "" && strings.TrimSpace(child.Deliverable) != "" {
			sb.WriteString("   - Deliverable: " + strings.TrimSpace(child.Deliverable) + "\n")
		}

		// If this child is in our path, process its children recursively
		if pathNodeIDs[child.ID] {
			if err := f.appendChildrenHierarchy(sb, child.ID, childPrefix, pathNodeIDs); err != nil {
				return err
			}
		}
	}

	return nil
}
