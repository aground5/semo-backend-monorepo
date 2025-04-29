package formatters

import (
	"strings"

	"semo-server/internal/models"
)

// UserDataFormatter formats user data for AI context
type UserDataFormatter struct{}

// NewUserDataFormatter creates a new UserDataFormatter
func NewUserDataFormatter() *UserDataFormatter {
	return &UserDataFormatter{}
}

// AppendUserTestsData appends user test data to a string builder
func (f *UserDataFormatter) AppendUserTestsData(builder *strings.Builder, userTests []models.UserTests) {
	if len(userTests) == 0 {
		return
	}

	// Add a separator if we already have data
	if builder.Len() > 0 {
		builder.WriteString("\n\n")
	}

	// Concatenate all user tests for this task
	for i, test := range userTests {
		if test.UserData != "" {
			if i > 0 {
				// Add separator between multiple tests for the same task
				builder.WriteString("\n")
			}
			builder.WriteString(test.UserData)
		}
	}
}

// FormatCombinedUserData formats a collection of user tests into a combined string
func (f *UserDataFormatter) FormatCombinedUserData(userTestsByTask [][]models.UserTests) string {
	var combinedData strings.Builder

	// Process each task's tests in order (parent -> child)
	for _, userTests := range userTestsByTask {
		// Append the user test data to our combined string
		f.AppendUserTestsData(&combinedData, userTests)
	}

	return combinedData.String()
}
