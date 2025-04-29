package models

// Subtask represents a single generated subtask
type Subtask struct {
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Goals       []string `json:"goals"`
	Objective   string   `json:"objective"`
	Deliverable string   `json:"deliverable"`
}

// SubtaskResponse represents the structured output from the subtask generation
type SubtaskResponse struct {
	Tasks          []Subtask `json:"tasks"`
	RedefinedTitle string    `json:"redefinedTitle"`
	RedefinedGoal  string    `json:"redefinedGoal"`
}

// EventType constants for different parts of the subtask output
const (
	EventTaskStart       = "task_start"
	EventTaskGoal        = "task_goal"
	EventTaskDeliverable = "task_deliverable"
	EventTaskEnd         = "task_end"
	EventRedefineTitle   = "redefine_title"
	EventRedefineGoal    = "redefine_goal"
	EventComplete        = "complete"
	EventError           = "error"
	EventRole            = "role"
)
