package models

// RoleResponse represents the structured output from the role generation
type RoleResponse struct {
	Think         string `json:"think"`
	Role          string `json:"role"`
	SystemMessage string `json:"systemMessage"`
}