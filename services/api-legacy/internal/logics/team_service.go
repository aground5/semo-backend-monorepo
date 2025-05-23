package logics

import (
	"errors"
	"fmt"
	"semo-server/configs-legacy"
	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"

	"gorm.io/gorm"
)

// TeamService handles team-related business logic
type TeamService struct {
	profileService *ProfileService
}

// NewTeamService creates a new instance of TeamService
func NewTeamService(profileSvc *ProfileService) *TeamService {
	return &TeamService{
		profileService: profileSvc,
	}
}

// GetTeamByID retrieves a team by its ID with optional relationships to preload
func (s *TeamService) GetTeamByID(id string, preloads ...string) (*models.Team, error) {
	var team models.Team

	// Start building the query
	query := repositories.DBS.Postgres

	// Add preloads if specified
	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	// Execute the query
	if err := query.First(&team, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("team not found")
		}
		return nil, err
	}

	return &team, nil
}

// CreateTeam creates a new team
func (s *TeamService) CreateTeam(name, description, photoURL string, creatorID string) (*models.Team, error) {
	// Generate a unique ID for the team
	teamID, err := utils.GenerateUniqueID("T")
	if err != nil {
		return nil, err
	}

	// Create the team
	team := models.Team{
		ID:          teamID,
		Name:        name,
		Description: description,
		Status:      "active",
		PhotoURL:    photoURL,
		CreatedBy:   creatorID,
	}

	// Start a transaction
	tx := repositories.DBS.Postgres.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	// Create the team
	if err := tx.Create(&team).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Make the creator a member and owner of the team
	memberID, err := utils.GenerateUniqueID("M")
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	membership := models.TeamMember{
		ID:        memberID,
		UserID:    creatorID,
		TeamID:    teamID,
		Role:      "owner",
		Status:    "active",
		InvitedBy: creatorID, // Creator invites themself
	}

	if err := tx.Create(&membership).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Return the created team with its creator
	return s.GetTeamByID(teamID, "Creator")
}

// UpdateTeam updates a team's information
func (s *TeamService) UpdateTeam(id string, updates models.TeamUpdate) (*models.Team, error) {
	// Get the team first
	team, err := s.GetTeamByID(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	updateMap := make(map[string]interface{})

	if updates.Name != nil {
		updateMap["name"] = *updates.Name
	}

	if updates.Description != nil {
		updateMap["description"] = *updates.Description
	}

	if updates.Status != nil {
		updateMap["status"] = *updates.Status
	}

	if updates.PhotoURL != nil {
		updateMap["photo_url"] = *updates.PhotoURL
	}

	// If there are updates to apply
	if len(updateMap) > 0 {
		if err := repositories.DBS.Postgres.Model(team).Updates(updateMap).Error; err != nil {
			return nil, err
		}
	}

	// Return the updated team
	return s.GetTeamByID(id)
}

// ListTeamMembers retrieves all members of a team with their roles
func (s *TeamService) ListTeamMembers(teamID string) ([]models.TeamMember, error) {
	var members []models.TeamMember

	if err := repositories.DBS.Postgres.Preload("Profile").Where("team_id = ?", teamID).Find(&members).Error; err != nil {
		return nil, err
	}

	return members, nil
}

// InviteUserToTeam sends an invitation to a user to join a team
func (s *TeamService) InviteUserToTeam(teamID, userID, inviterID, role string) error {
	// Verify the team exists
	team, err := s.GetTeamByID(teamID)
	if err != nil {
		return err
	}

	// Verify the user exists
	profiles, err := s.profileService.GetProfilesByIDs([]string{userID, inviterID})
	if err != nil {
		return err
	}
	if len(profiles) < 2 {
		return errors.New("user or inviter not found")
	}

	var userProfile, inviterProfile models.Profile
	for _, profile := range profiles {
		if profile.ID == userID {
			userProfile = profile
		}
		if profile.ID == inviterID {
			inviterProfile = profile
		}
	}

	if userProfile.ID == "" {
		return errors.New("user not found")
	}
	if inviterProfile.ID == "" {
		return errors.New("inviter not found")
	}

	// Check if the user is already a member
	var existingMembership models.TeamMember
	err = repositories.DBS.Postgres.Where("team_id = ? AND user_id = ?", teamID, userID).First(&existingMembership).Error

	// Generate a unique ID for the membership if needed
	var membershipID string

	// If the membership already exists but is not in 'active' status, update it
	if err == nil {
		if existingMembership.Status == "active" {
			return errors.New("user is already a member of this team")
		}

		// Update the existing membership
		membershipID = existingMembership.ID
		err = repositories.DBS.Postgres.Model(&existingMembership).Updates(map[string]interface{}{
			"status":     "invited",
			"role":       role,
			"invited_by": inviterID,
		}).Error

		if err != nil {
			return err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// If there was an error other than not found
		return err
	} else {
		// Generate a unique ID for the new membership
		membershipID, err = utils.GenerateUniqueID("M")
		if err != nil {
			return err
		}

		// Create a new team membership
		membership := models.TeamMember{
			ID:        membershipID,
			UserID:    userID,
			TeamID:    teamID,
			Role:      role,
			Status:    "invited", // Start as invited
			InvitedBy: inviterID,
		}

		if err := repositories.DBS.Postgres.Create(&membership).Error; err != nil {
			return err
		}
	}

	// Send an email invitation
	inviteURL := fmt.Sprintf("%s/teams/%s/invitations/%s", configs.Configs.Service.BaseURL, teamID, membershipID)

	err = utils.EmailSvc.SendTeamInvitationEmail(
		configs.Configs.Email.SenderEmail,
		userProfile.Email,
		userProfile.Name,
		team.Name,
		inviterProfile.Name,
		inviteURL,
	)

	if err != nil {
		// Log the error but don't fail the invitation process
		fmt.Printf("Failed to send invitation email: %v\n", err)
	}

	return nil
}

// AcceptTeamInvitation lets a user accept an invitation to join a team
func (s *TeamService) AcceptTeamInvitation(membershipID string) error {
	var membership models.TeamMember

	// Find the invitation
	if err := repositories.DBS.Postgres.First(&membership, "id = ? AND status = ?", membershipID, "invited").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invitation not found or already processed")
		}
		return err
	}

	// Update the status to active
	return repositories.DBS.Postgres.Model(&membership).Update("status", "active").Error
}

// RejectTeamInvitation lets a user reject an invitation to join a team
func (s *TeamService) RejectTeamInvitation(membershipID string) error {
	var membership models.TeamMember

	// Find the invitation
	if err := repositories.DBS.Postgres.First(&membership, "id = ? AND status = ?", membershipID, "invited").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invitation not found or already processed")
		}
		return err
	}

	// Update the status to rejected
	return repositories.DBS.Postgres.Model(&membership).Update("status", "rejected").Error
}

// RemoveUserFromTeam removes a user from a team
func (s *TeamService) RemoveUserFromTeam(teamID, userID string) error {
	var membership models.TeamMember

	// Find the membership
	if err := repositories.DBS.Postgres.Where("team_id = ? AND user_id = ? AND status = ?", teamID, userID, "active").First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user is not an active member of this team")
		}
		return err
	}

	// Soft delete the membership
	return repositories.DBS.Postgres.Delete(&membership).Error
}

// UpdateMemberRole updates a team member's role
func (s *TeamService) UpdateMemberRole(teamID, userID, newRole string) error {
	var membership models.TeamMember

	// Find the membership
		if err := repositories.DBS.Postgres.Where("team_id = ? AND user_id = ? AND status = ?", teamID, userID, "active").First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user is not an active member of this team")
		}
		return err
	}

	// Update the role
	return repositories.DBS.Postgres.Model(&membership).Update("role", newRole).Error
}

// ListUserInvitations retrieves all pending invitations for a user
func (s *TeamService) ListUserInvitations(userID string) ([]models.TeamMember, error) {
	var invitations []models.TeamMember

	if err := repositories.DBS.Postgres.Preload("Team").Preload("Inviter").
		Where("user_id = ? AND status = ?", userID, "invited").
		Find(&invitations).Error; err != nil {
		return nil, err
	}

	return invitations, nil
}
