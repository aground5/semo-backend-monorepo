package formatters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

// DataFormatter formats data for prompts
type DataFormatter struct{}

// NewDataFormatter creates a new DataFormatter
func NewDataFormatter() *DataFormatter {
	return &DataFormatter{}
}

// FormatPrompt formats a prompt with template variables
func (f *DataFormatter) FormatPrompt(promptText string, params map[string]any) (string, error) {
	tmpl, err := template.New("prompt").Funcs(template.FuncMap{
		"sub": func(a, b int) int { return a - b },
	}).Parse(promptText)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// FormatTasksAsJSON formats tasks as JSON for API requests
func (f *DataFormatter) FormatTasksAsJSON(tasks []map[string]interface{}) (string, error) {
	// Create the JSON data
	jsonData := map[string]interface{}{
		"tasks": tasks,
	}

	// Convert to JSON string
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tasks to JSON: %w", err)
	}

	return string(jsonBytes), nil
}
