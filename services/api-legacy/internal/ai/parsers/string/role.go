package stringparser

import (
	"fmt"
	"regexp"
	"strings"

	"semo-server/internal/ai/models"
)

// RoleParser parses role responses
type RoleParser struct{}

// NewRoleParser creates a new RoleParser
func NewRoleParser() *RoleParser {
	return &RoleParser{}
}

// GetType returns the type of parser
func (p *RoleParser) GetType() string {
	return "role_string_parser"
}

// Parse parses the output from the role generation
func (p *RoleParser) Parse(llmOutput string) (*models.RoleResponse, error) {
	// This pattern uses:
	//   - (?s) to allow the dot to match newline characters
	//   - (.*?) to lazily capture text until the next piece
	//   - the explicit "Think:", "Role:", and "System message:" markers
	pattern := `(?s)Think:\s*(.*?)\s*Role:\s*(.*?)\s*System message:\s*(.*)`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(llmOutput)
	if len(matches) < 4 {
		return nil, fmt.Errorf("input string does not match the expected format")
	}

	return &models.RoleResponse{
		Think:         strings.TrimSpace(matches[1]),
		Role:          strings.TrimSpace(matches[2]),
		SystemMessage: strings.TrimSpace(matches[3]),
	}, nil
}