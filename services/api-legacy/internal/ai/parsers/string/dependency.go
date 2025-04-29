package stringparser

import (
	"encoding/json"
	"fmt"

	"semo-server/internal/ai/models"
)

// DependencyParser parses task dependency responses
type DependencyParser struct{}

// NewDependencyParser creates a new DependencyParser
func NewDependencyParser() *DependencyParser {
	return &DependencyParser{}
}

// GetType returns the type of parser
func (p *DependencyParser) GetType() string {
	return "dependency_string_parser"
}

// Parse parses the task dependency response
func (p *DependencyParser) Parse(llmOutput string) (*models.TaskDependencyResponse, error) {
	// Find the JSON content in the response
	// Look for patterns like ```json ... ``` or just a JSON object
	var jsonContent string

	// First check if the response contains a JSON block
	jsonStartIdx := -1
	jsonEndIdx := -1

	// Look for ```json or ``` pattern
	jsonBlockStart := []string{"```json", "```"}
	jsonBlockEnd := "```"

	for _, startMarker := range jsonBlockStart {
		startIdx := -1
		if start := findIndexWithTrim(llmOutput, startMarker); start != -1 {
			startIdx = start + len(startMarker)
			if end := findIndexWithTrim(llmOutput[startIdx:], jsonBlockEnd); end != -1 {
				jsonStartIdx = startIdx
				jsonEndIdx = startIdx + end
				break
			}
		}
	}

	// Extract the JSON content
	if jsonStartIdx != -1 && jsonEndIdx != -1 {
		jsonContent = llmOutput[jsonStartIdx:jsonEndIdx]
	} else {
		// If no code block markers found, try to find a JSON object directly
		openBrace := findIndexWithTrim(llmOutput, "{")
		if openBrace != -1 {
			closeBrace := findLastIndexWithTrim(llmOutput, "}")
			if closeBrace > openBrace {
				jsonContent = llmOutput[openBrace : closeBrace+1]
			}
		}
	}

	// If we couldn't extract JSON content, return an error
	if jsonContent == "" {
		return nil, fmt.Errorf("could not extract JSON content from LLM output")
	}

	// Parse the JSON content
	var response models.TaskDependencyResponse
	if err := json.Unmarshal([]byte(jsonContent), &response); err != nil {
		return nil, fmt.Errorf("failed to parse task dependency JSON: %w", err)
	}

	return &response, nil
}

// Helper functions for parsing the LLM output

// findIndexWithTrim finds the first occurrence of a substring with trimming
func findIndexWithTrim(s, substr string) int {
	return indexOfTrimmed(s, substr, true)
}

// findLastIndexWithTrim finds the last occurrence of a substring with trimming
func findLastIndexWithTrim(s, substr string) int {
	return indexOfTrimmed(s, substr, false)
}

// indexOfTrimmed is a helper function to find index with trimming, first or last occurrence
func indexOfTrimmed(s, substr string, first bool) int {
	lines := splitLines(s)

	for i, line := range lines {
		trimmedLine := line
		idx := -1

		if first {
			idx = indexOf(trimmedLine, substr)
		} else {
			idx = lastIndexOf(trimmedLine, substr)
		}

		if idx != -1 {
			// Calculate the absolute position in the original string
			pos := 0
			for j := 0; j < i; j++ {
				pos += len(lines[j]) + 1 // +1 for the newline
			}
			return pos + idx
		}
	}

	return -1
}

// indexOf finds the first occurrence of substr in s
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// lastIndexOf finds the last occurrence of substr in s
func lastIndexOf(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
