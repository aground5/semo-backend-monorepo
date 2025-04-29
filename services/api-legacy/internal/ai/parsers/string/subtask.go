package stringparser

import (
	"regexp"
	"strconv"
	"strings"

	"semo-server/internal/ai/models"
)

// SubtaskParser parses subtask responses
type SubtaskParser struct{}

// NewSubtaskParser creates a new SubtaskParser
func NewSubtaskParser() *SubtaskParser {
	return &SubtaskParser{}
}

// GetType returns the type of parser
func (p *SubtaskParser) GetType() string {
	return "subtask_string_parser"
}

// Parse parses the complete output from the subtask generation
// and returns a structured representation
func (p *SubtaskParser) Parse(output string) (*models.SubtaskResponse, error) {
	// Split the output into the subtask section and the redefine section
	parts := strings.Split(output, "Redefine To-do:")

	subtasksSection := ""
	redefineSection := ""

	if len(parts) == 2 {
		subtasksSection = strings.TrimSpace(parts[0])
		redefineSection = strings.TrimSpace(parts[1])
	} else {
		subtasksSection = strings.TrimSpace(output)
	}

	// Check if there's a "Sub-to-do:" section
	parts = strings.Split(subtasksSection, "Sub-to-do:")
	if len(parts) > 1 {
		_ = strings.TrimSpace(parts[0])
		subtasksSection = strings.TrimSpace(parts[1])
	} else {
		subtasksSection = strings.TrimSpace(parts[0])
	}

	// Parse the subtasks
	lines := strings.Split(subtasksSection, "\n")

	var result models.SubtaskResponse
	var currentTask *models.Subtask

	// Regular expressions for parsing subtasks
	taskTitlePattern := regexp.MustCompile(`^\s*(\d+)\.\s+(.+)$`)
	taskGoalPattern := regexp.MustCompile(`^\s*-\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a task title line
		if taskMatches := taskTitlePattern.FindStringSubmatch(line); len(taskMatches) > 2 {
			// If we were processing a task, append it to our results
			if currentTask != nil {
				result.Tasks = append(result.Tasks, *currentTask)
			}

			// Start a new task
			taskNumber, _ := strconv.Atoi(taskMatches[1])
			currentTask = &models.Subtask{
				Number: taskNumber,
				Title:  taskMatches[2],
				Goals:  []string{},
			}
		} else if goalMatches := taskGoalPattern.FindStringSubmatch(line); len(goalMatches) > 1 && currentTask != nil {
			// This is a goal line, add it to the current task
			currentTask.Goals = append(currentTask.Goals, goalMatches[1])
		}
	}

	// Add the last task if we were processing one
	if currentTask != nil {
		result.Tasks = append(result.Tasks, *currentTask)
	}

	// Parse the redefined title and goal
	// The first line is the title, and the rest is the goal
	if redefineSection != "" {
		redefineLines := strings.Split(redefineSection, "\n")
		if len(redefineLines) > 0 {
			result.RedefinedTitle = redefineLines[0]

			if len(redefineLines) > 1 {
				result.RedefinedGoal = strings.Join(redefineLines[1:], " ")
				result.RedefinedGoal = strings.TrimSpace(result.RedefinedGoal)
			}
		}
	}

	return &result, nil
}
