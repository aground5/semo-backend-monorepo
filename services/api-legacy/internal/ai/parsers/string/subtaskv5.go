package stringparser

import (
	"regexp"
	"strconv"
	"strings"

	"semo-server/internal/ai/models"
)

// SubtaskV5Parser parses subtask v5 responses
type SubtaskV5Parser struct{}

// NewSubtaskV5Parser creates a new SubtaskV5Parser
func NewSubtaskV5Parser() *SubtaskV5Parser {
	return &SubtaskV5Parser{}
}

// GetType returns the type of parser
func (p *SubtaskV5Parser) GetType() string {
	return "subtask_v5_string_parser"
}

// Parse parses the complete output from the subtask v5 generation
// and returns a structured representation
func (p *SubtaskV5Parser) Parse(output string) (*models.SubtaskResponse, error) {
	// Create the result structure
	var result models.SubtaskResponse

	// Extract content within final_answer tags if present
	finalAnswerPattern := regexp.MustCompile(`(?s)<final_answer>(.*?)</final_answer>`)
	matches := finalAnswerPattern.FindStringSubmatch(output)
	if len(matches) > 1 {
		output = matches[1]
	}

	// Extract the main todo
	todoPattern := regexp.MustCompile(`(?i)Todo:\s*(.+)`)
	todoMatches := todoPattern.FindStringSubmatch(output)
	if len(todoMatches) > 1 {
		result.RedefinedTitle = strings.TrimSpace(todoMatches[1])
	}

	// Split into sections
	sections := strings.Split(output, "Sub-Todos:")
	if len(sections) < 2 {
		// No subtasks section found, return just the redefined title
		return &result, nil
	}

	// Process the subtasks section
	subtasksSection := sections[1]

	// Regular expressions for parsing tasks
	taskPattern := regexp.MustCompile(`(?m)^\s*(\d+)\.\s+(.+)$`)
	objectivePattern := regexp.MustCompile(`(?i)-\s*Objective:\s*(.+)`)
	deliverablePattern := regexp.MustCompile(`(?i)-\s*Deliverable:\s*(.+)`)

	// Find all task titles with their numbers
	taskMatches := taskPattern.FindAllStringSubmatchIndex(subtasksSection, -1)

	for i, taskMatch := range taskMatches {
		// Extract the task number and title
		taskNumber, _ := strconv.Atoi(subtasksSection[taskMatch[2]:taskMatch[3]])
		taskTitle := strings.TrimSpace(subtasksSection[taskMatch[4]:taskMatch[5]])

		// Determine the content range for this task
		startPos := taskMatch[1]
		endPos := len(subtasksSection)
		if i+1 < len(taskMatches) {
			endPos = taskMatches[i+1][0]
		}

		// Extract the task content
		taskContent := subtasksSection[startPos:endPos]

		// Extract objective
		objectiveMatches := objectivePattern.FindStringSubmatch(taskContent)
		objective := ""
		if len(objectiveMatches) > 1 {
			objective = strings.TrimSpace(objectiveMatches[1])
		}

		// Extract deliverable
		deliverableMatches := deliverablePattern.FindStringSubmatch(taskContent)
		deliverable := ""
		if len(deliverableMatches) > 1 {
			deliverable = strings.TrimSpace(deliverableMatches[1])
		}

		// Create the subtask
		subtask := models.Subtask{
			Number:      taskNumber,
			Title:       taskTitle,
			Goals:       []string{},
			Objective:   objective,
			Deliverable: deliverable,
		}

		// Add objectives and deliverables as goals
		if objective != "" {
			subtask.Goals = append(subtask.Goals, objective)
		}
		if deliverable != "" {
			subtask.Goals = append(subtask.Goals, deliverable)
		}

		// Add to result
		result.Tasks = append(result.Tasks, subtask)
	}

	return &result, nil
}
