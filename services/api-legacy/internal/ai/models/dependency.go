package models

// TaskDependency represents an individual task with its dependencies
type TaskDependency struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Dependencies []string `json:"dependencies"`
}

// TaskDependencyResponse represents a task with its dependencies
type TaskDependencyResponse struct {
	Tasks []TaskDependency `json:"tasks"`
}