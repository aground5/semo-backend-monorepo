package utils

import (
	"fmt"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// UniqueIDService provides ID generation functionality
type UniqueIDService struct{}

// NewUniqueIDService creates a new UniqueIDService
func NewUniqueIDService() *UniqueIDService {
	return &UniqueIDService{}
}

// GenerateID creates an ID with the following pattern:
//   - First character is the provided prefix (e.g., 'o' for organization)
//   - Followed by 2 random digits [0-9]
//   - Followed by 9 random alphanumeric [0-9a-z]
//
// Example output with prefix 'o': o12abc345xy
func (s *UniqueIDService) GenerateID(prefix string) (string, error) {
	// Define our alphabets
	digits := "0123456789"
	alnum := "0123456789abcdefghijklmnopqrstuvwxyz"

	// Generate 2 digits
	twoDigits, err := gonanoid.Generate(digits, 2)
	if err != nil {
		return "", fmt.Errorf("failed to generate two digits: %w", err)
	}

	// Generate 9 alphanumeric chars
	nineAlnum, err := gonanoid.Generate(alnum, 9)
	if err != nil {
		return "", fmt.Errorf("failed to generate alphanumeric part: %w", err)
	}

	// Concatenate to form the final ID
	return strings.ToUpper(prefix + twoDigits + nineAlnum), nil
}

// GenerateRandomColor creates a random 6-digit hex color code.
// Example output: "A1B2C3"
func (s *UniqueIDService) GenerateRandomColor() (string, error) {
	// Hexadecimal digits (소문자로 생성 후 최종 결과에서 대문자로 변환)
	hexDigits := "0123456789abcdef"

	// Generate 6 random hex characters
	color, err := gonanoid.Generate(hexDigits, 6)
	if err != nil {
		return "", fmt.Errorf("failed to generate random color: %w", err)
	}

	// 문자열을 대문자로 변환하여 반환
	return strings.ToUpper(color), nil
}

// Global instance of UniqueIDService (기존 코드와의 호환성을 위해 유지)
var UniqueIDSvc = NewUniqueIDService()

// Compatibility function for existing code
func GenerateUniqueID(prefix string) (string, error) {
	return UniqueIDSvc.GenerateID(prefix)
}
