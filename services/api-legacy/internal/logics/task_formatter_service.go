package logics

import (
	"encoding/json"
	"fmt"
	"strings"

	"semo-server/internal/models"

	"gorm.io/gorm"
)

// TaskFormatterService handles the formatting of tasks in various representations
type TaskFormatterService struct {
	itemService *ItemService
	db          *gorm.DB
}

// TaskJSON represents a task in JSON format
type TaskJSON struct {
	ID        string `json:"id"`
	Numbering string `json:"numbering"`
	Name      string `json:"name"`
	Goal      string `json:"goal"`
	ParentID  string `json:"parent_id,omitempty"`
	Level     int    `json:"level,omitempty"` // Task depth level (optional)
}

// TasksJSON represents a collection of tasks in JSON format
type TasksJSON struct {
	Tasks []TaskJSON `json:"tasks"`
}

// TaskHierarchy represents a structured view of a task and its subtasks
type TaskHierarchy struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Contents string           `json:"contents,omitempty"`
	Type     string           `json:"type"`
	Children []*TaskHierarchy `json:"children,omitempty"`
}

// NewTaskFormatterService creates a new instance of TaskFormatterService
func NewTaskFormatterService(db *gorm.DB, itemService *ItemService) *TaskFormatterService {
	return &TaskFormatterService{
		db:          db,
		itemService: itemService,
	}
}

// GetFormattedTaskFromID retrieves a task by ID and returns it in a formatted representation
// If max_level is >= 0, only tasks up to that level in the hierarchy will be included
// If max_level is < 0 (default), all levels will be included
func (ts *TaskFormatterService) GetFormattedTaskFromID(taskID string, maxLevel ...int) (string, error) {
	// Default max level to -1 (unlimited depth)
	max := -1
	if len(maxLevel) > 0 {
		max = maxLevel[0]
	}

	// Get the initial task
	task, err := ts.itemService.GetItem(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get task with ID %s: %w", taskID, err)
	}

	// Find the root parent by traversing up the parent hierarchy
	rootTask := task
	for rootTask.ParentID != nil && *rootTask.ParentID != "" {
		parentTask, err := ts.itemService.GetItem(*rootTask.ParentID)
		if err != nil {
			return "", fmt.Errorf("failed to get parent task with ID %s: %w", *rootTask.ParentID, err)
		}
		rootTask = parentTask
	}

	// Build the numbered representation of the entire task tree.
	// Root task starts with number chain [1]
	var sb strings.Builder
	if err := ts.buildTaskTreeMarkdown(&sb, rootTask, []int{1}, 0, max); err != nil {
		return "", fmt.Errorf("failed to build task tree markdown: %w", err)
	}

	return strings.TrimSpace(sb.String()), nil
}

// buildTaskTreeMarkdown recursively builds a numbered representation of a task tree.
// 'numbering' parameter represents the number chain of the current task, e.g., [1,2] means "1.2."
// 'currentLevel' tracks the current depth in the task hierarchy
// 'maxLevel' specifies the maximum depth to include (-1 for unlimited)
func (ts *TaskFormatterService) buildTaskTreeMarkdown(sb *strings.Builder, task *models.Item, numbering []int, currentLevel int, maxLevel int) error {
	// Convert number chain to string (e.g., "1.1.2.")
	var numStr strings.Builder
	if len(numbering) == 1 {
		numStr.WriteString(fmt.Sprintf("#"))
	}
	for idx, num := range numbering {
		if idx > 0 {
			numStr.WriteString(fmt.Sprintf("%d.", num))
		}
	}

	// Output task title
	sb.WriteString(fmt.Sprintf("%s %s\n", numStr.String(), task.Name))

	// Output task description (Contents) if available
	if task.Contents != "" && strings.TrimSpace(task.Contents) != "" {
		sb.WriteString(task.Contents + "\n")
	}

	// If we've reached the max level and it's not unlimited (-1), don't process children
	if maxLevel >= 0 && currentLevel >= maxLevel {
		return nil
	}

	// Get child tasks
	var subtasks []models.Item
	if err := ts.db.Order("position ASC").Where("parent_id = ?", task.ID).Find(&subtasks).Error; err != nil {
		return err
	}

	// Process each child task recursively
	for i, subtask := range subtasks {
		// Add empty line between tasks (even for the first child if parent has description)
		if i > 0 || task.Contents != "" {
			sb.WriteString("\n")
		}

		// Create new number chain by appending child index (i+1) to the current chain
		newNumbering := append(append([]int(nil), numbering...), i+1)
		if err := ts.buildTaskTreeMarkdown(sb, &subtask, newNumbering, currentLevel+1, maxLevel); err != nil {
			return err
		}
	}

	return nil
}

// GetTaskHierarchy returns a hierarchical representation of tasks (can be customized for different formats)
func (ts *TaskFormatterService) GetTaskHierarchy(taskID string) (*TaskHierarchy, error) {
	// This is a placeholder for a function that could return a structured representation
	// of the task hierarchy instead of a markdown string
	task, err := ts.itemService.GetItem(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task with ID %s: %w", taskID, err)
	}

	// Find the root task
	rootTask := task
	for rootTask.ParentID != nil && *rootTask.ParentID != "" {
		parentTask, err := ts.itemService.GetItem(*rootTask.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent task with ID %s: %w", *rootTask.ParentID, err)
		}
		rootTask = parentTask
	}

	// Build the hierarchy
	hierarchy, err := ts.buildTaskHierarchy(rootTask)
	if err != nil {
		return nil, fmt.Errorf("failed to build task hierarchy: %w", err)
	}

	return hierarchy, nil
}

// GetTasksAsJSON returns a JSON string representation of the task hierarchy
func (ts *TaskFormatterService) GetTasksAsJSON(taskID string) (string, error) {
	// Create the TasksJSON structure
	tasksJSON := TasksJSON{
		Tasks: []TaskJSON{},
	}

	// Get the task and all its children
	task, err := ts.itemService.GetItem(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get task with ID %s: %w", taskID, err)
	}

	// Find the root task
	rootTask := task
	for rootTask.ParentID != nil && *rootTask.ParentID != "" {
		parentTask, err := ts.itemService.GetItem(*rootTask.ParentID)
		if err != nil {
			return "", fmt.Errorf("failed to get parent task with ID %s: %w", *rootTask.ParentID, err)
		}
		rootTask = parentTask
	}

	// Process task hierarchy into flattened JSON format
	if err := ts.buildTaskJSONList(&tasksJSON.Tasks, rootTask, "", map[string]bool{}); err != nil {
		return "", fmt.Errorf("failed to build task JSON: %w", err)
	}

	// Convert to JSON string
	jsonBytes, err := json.MarshalIndent(tasksJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal tasks to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// buildTaskJSONList recursively processes the task hierarchy and builds a flat list of tasks in JSON format
func (ts *TaskFormatterService) buildTaskJSONList(tasks *[]TaskJSON, task *models.Item, prefix string, visited map[string]bool) error {
	return ts.buildTaskJSONListWithLevel(tasks, task, prefix, 0, visited)
}

// buildTaskJSONListWithLevel recursively processes the task hierarchy with level information
func (ts *TaskFormatterService) buildTaskJSONListWithLevel(tasks *[]TaskJSON, task *models.Item, prefix string, level int, visited map[string]bool) error {
	// Avoid processing the same task twice (prevents cycles)
	if visited[task.ID] {
		return nil
	}
	visited[task.ID] = true

	// Create task ID with prefix
	taskID := prefix
	if prefix == "" {
		// For root tasks, just use sequence numbers
		taskID = "1"
	}

	// Extract goal from Contents (if it exists)
	goal := task.Contents
	if goal == "" {
		goal = "No goal specified"
	}

	// Add this task to the JSON list
	taskJSON := TaskJSON{
		ID:        task.ID,
		Numbering: taskID,
		Name:      task.Name,
		Goal:      goal,
		Level:     level,
	}

	// Set ParentID if this is not a root task
	if task.ParentID != nil && *task.ParentID != "" {
		// Find parent ID from task relationships
		parentTask, err := ts.itemService.GetItem(*task.ParentID)
		if err != nil {
			return err
		}

		// Find the parent's formatted ID in our tasks list
		for _, t := range *tasks {
			if t.Name == parentTask.Name {
				taskJSON.ParentID = t.ID
				break
			}
		}
	}

	*tasks = append(*tasks, taskJSON)

	// Get child tasks
	var subtasks []models.Item
	if err := ts.db.Order("position ASC").Where("parent_id = ?", task.ID).Find(&subtasks).Error; err != nil {
		return err
	}

	// Process each child task recursively
	for i, subtask := range subtasks {
		// Create new task ID prefix for children
		childPrefix := fmt.Sprintf("%s.%d", taskID, i+1)
		if err := ts.buildTaskJSONListWithLevel(tasks, &subtask, childPrefix, level+1, visited); err != nil {
			return err
		}
	}

	return nil
}

// GetFormattedTaskAsJSON is a convenience method that returns the JSON for a specific task
func (ts *TaskFormatterService) GetFormattedTaskAsJSON(taskID string) (string, error) {
	return ts.GetTasksAsJSON(taskID)
}

// GetFormattedTaskAsJSONWithOptions returns a JSON representation of tasks with configurable options
func (ts *TaskFormatterService) GetFormattedTaskAsJSONWithOptions(taskID string, options map[string]interface{}) (string, error) {
	// Create the TasksJSON structure
	tasksJSON := TasksJSON{
		Tasks: []TaskJSON{},
	}

	// Get the task
	task, err := ts.itemService.GetItem(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get task with ID %s: %w", taskID, err)
	}

	// Handle different traversal options
	traversalMode := "full" // Default to full traversal
	if mode, ok := options["traversal_mode"].(string); ok {
		traversalMode = mode
	}

	// Handle specific level filtering
	var specificLevel int = -1 // Default to no specific level
	if level, ok := options["level"].(int); ok {
		specificLevel = level
	}

	var rootTask *models.Item

	switch traversalMode {
	case "self_only":
		// Only include the task itself, no parents or children
		rootTask = task
	case "with_children":
		// Include the task and its direct children
		rootTask = task
	case "with_ancestors":
		// Include the task, its ancestors, but not siblings
		rootTask = ts.findRootTask(task)
	case "same_level":
		// For same_level, we'll handle it separately
		rootTask = task
	case "specific_level":
		// Include tasks at a specific level only
		rootTask = ts.findRootTask(task)
	case "full":
		// Default: find the root task and include the entire hierarchy
		rootTask = ts.findRootTask(task)
	default:
		rootTask = ts.findRootTask(task)
	}

	// Special handling for same_level mode
	if traversalMode == "same_level" {
		// Check if the task is a root task (has no parent)
		if task.ParentID == nil || *task.ParentID == "" {
			// If it's a root task, only return itself
			taskJSON := TaskJSON{
				ID:        task.ID,
				Numbering: "1",
				Name:      task.Name,
				Goal:      task.Contents,
				Level:     0,
			}
			tasksJSON.Tasks = append(tasksJSON.Tasks, taskJSON)
		} else {
			// Get all tasks with the same parent ID
			var sameLevelTasks []models.Item
			if err := ts.db.Order("position ASC").Where("parent_id = ?", *task.ParentID).Find(&sameLevelTasks).Error; err != nil {
				return "", fmt.Errorf("failed to fetch same level tasks: %w", err)
			}

			// Convert to TaskJSON format
			for i, sameTask := range sameLevelTasks {
				numbering := fmt.Sprintf("%d", i+1)
				taskJSON := TaskJSON{
					ID:        sameTask.ID,
					Numbering: numbering,
					Name:      sameTask.Name,
					Goal:      sameTask.Contents,
					Level:     0, // Same level tasks are assigned level 0
					ParentID:  *task.ParentID,
				}
				tasksJSON.Tasks = append(tasksJSON.Tasks, taskJSON)
			}
		}

		// Convert to JSON string
		jsonBytes, err := json.MarshalIndent(tasksJSON, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal tasks to JSON: %w", err)
		}

		return string(jsonBytes), nil
	}

	// Special handling for specific_level mode
	if traversalMode == "specific_level" && specificLevel >= 0 {
		// We need to build the full hierarchy first to identify levels
		visited := make(map[string]bool)
		if err := ts.buildTaskJSONListWithLevel(&tasksJSON.Tasks, rootTask, "", 0, visited); err != nil {
			return "", fmt.Errorf("failed to build task JSON: %w", err)
		}

		// Filter tasks to include only those at the specified level
		var filteredTasks []TaskJSON
		for _, t := range tasksJSON.Tasks {
			if t.Level == specificLevel {
				filteredTasks = append(filteredTasks, t)
			}
		}
		tasksJSON.Tasks = filteredTasks

		// Convert to JSON string
		jsonBytes, err := json.MarshalIndent(tasksJSON, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal tasks to JSON: %w", err)
		}

		return string(jsonBytes), nil
	}

	// For other modes, build the task hierarchy
	visited := make(map[string]bool)

	if traversalMode == "self_only" {
		// Add just this task
		taskJSON := TaskJSON{
			ID:        task.ID,
			Numbering: "1",
			Name:      task.Name,
			Goal:      task.Contents,
			Level:     0,
		}
		if task.ParentID != nil && *task.ParentID != "" {
			taskJSON.ParentID = *task.ParentID // Use the actual parent ID
		}
		tasksJSON.Tasks = append(tasksJSON.Tasks, taskJSON)
	} else {
		// Build the full task list first
		if err := ts.buildTaskJSONListWithLevel(&tasksJSON.Tasks, rootTask, "", 0, visited); err != nil {
			return "", fmt.Errorf("failed to build task JSON: %w", err)
		}

		// Filter based on traversal mode
		if traversalMode == "with_children" {
			// Keep only the task and its direct children
			var filteredTasks []TaskJSON
			rootID := rootTask.ID
			for _, t := range tasksJSON.Tasks {
				if t.ParentID == rootID || t.ID == rootID {
					filteredTasks = append(filteredTasks, t)
				}
			}
			tasksJSON.Tasks = filteredTasks
		}
	}

	// Convert to JSON string
	jsonBytes, err := json.MarshalIndent(tasksJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal tasks to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// findRootTask traverses up the parent hierarchy to find the root task
func (ts *TaskFormatterService) findRootTask(task *models.Item) *models.Item {
	rootTask := task
	for rootTask.ParentID != nil && *rootTask.ParentID != "" {
		parentTask, err := ts.itemService.GetItem(*rootTask.ParentID)
		if err != nil {
			// If we can't get the parent, just return the current task as root
			return rootTask
		}
		rootTask = parentTask
	}
	return rootTask
}

// buildTaskHierarchy recursively builds a structured representation of a task hierarchy
func (ts *TaskFormatterService) buildTaskHierarchy(task *models.Item) (*TaskHierarchy, error) {
	hierarchy := &TaskHierarchy{
		ID:       task.ID,
		Name:     task.Name,
		Contents: task.Contents,
		Type:     task.Type,
		Children: []*TaskHierarchy{},
	}

	// Get child tasks
	var subtasks []models.Item
	if err := ts.db.Order("position ASC").Where("parent_id = ?", task.ID).Find(&subtasks).Error; err != nil {
		return nil, err
	}

	// Process each child task recursively
	for _, subtask := range subtasks {
		childHierarchy, err := ts.buildTaskHierarchy(&subtask)
		if err != nil {
			return nil, err
		}
		hierarchy.Children = append(hierarchy.Children, childHierarchy)
	}

	return hierarchy, nil
}
