package logics

import (
	"errors"
	"fmt"

	"authn-server/internal/models"
	"authn-server/internal/repositories"
)

// InitialService handles initial setup logic for users and organizations
type InitialService struct{}

// NewInitialService creates a new InitialService
func NewInitialService() *InitialService {
	return &InitialService{}
}

// GetUserName retrieves a user's name by ID
func (s *InitialService) GetUserName(userID string) (string, error) {
	var user models.User
	if err := repositories.DBS.Postgres.Find(&user, "id = ?", userID).Error; err != nil {
		return "", err
	}
	return user.Name, nil
}

// UpdateUserName updates a user's name
func (s *InitialService) UpdateUserName(userID, newName string) error {
	if newName == "" {
		return errors.New("name cannot be empty")
	}
	result := repositories.DBS.Postgres.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("name", newName)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user with id %s not found", userID)
	}
	return nil
}

// CreateOrganization creates a new organization
// Required parameter: name
// Defaults: plan="free", i18nLanguage="ko", PhotoURL=""
func (s *InitialService) CreateOrganization(name, userID string) (*models.Organization, error) {
	if name == "" {
		return nil, errors.New("organization name is required")
	}

	// Generate unique organization ID
	orgID, err := GenerateUniqueID("o")
	if err != nil {
		return nil, fmt.Errorf("failed to generate organization ID: %w", err)
	}

	org := &models.Organization{
		ID:           orgID,
		Name:         name,
		Plan:         "free", // Default value
		I18nLanguage: "ko",   // Default value
		PhotoURL:     "",     // Default value
		Status:       "requested",
		CreatedBy:    userID,
	}

	if err := repositories.DBS.Postgres.Create(org).Error; err != nil {
		return nil, err
	}
	return org, nil
}

// NeedChangeName checks if a user needs to set their name
func (s *InitialService) NeedChangeName(userID string) (bool, error) {
	var user models.User
	if err := repositories.DBS.Postgres.
		Select("name").
		Where("id = ?", userID).
		First(&user).Error; err != nil {
		return false, err
	}
	return user.Name == "", nil
}

// NeedCreateOrganization checks if a user needs to create an organization
func (s *InitialService) NeedCreateOrganization(userID string) (bool, error) {
	var count int64
	if err := repositories.DBS.Postgres.
		Model(&models.Organization{}).
		Where("created_by = ?", userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

// Global instance of InitialService
var InitialSvc = NewInitialService()
