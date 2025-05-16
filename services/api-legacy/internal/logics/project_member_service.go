package logics

import (
	"errors"
	"fmt"
	"semo-server/internal/models"
	"semo-server/internal/repositories"

	"gorm.io/gorm"
)

// ProjectMemberService handles project member-related business logic
type ProjectMemberService struct {
	teamService    *TeamService
	profileService *ProfileService
}

// NewProjectMemberService creates a new instance of ProjectMemberService
func NewProjectMemberService(profileService *ProfileService, teamService *TeamService) *ProjectMemberService {
	return &ProjectMemberService{
		teamService:    teamService,
		profileService: profileService,
	}
}

// GetProjectTeam retrieves the team associated with a project
func (s *ProjectMemberService) GetProjectTeam(projectID string) (*models.Team, error) {
	var team models.Team
	if err := repositories.DBS.Postgres.Where("project_id = ?", projectID).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no team found for project with ID %s", projectID)
		}
		return nil, fmt.Errorf("failed to retrieve team for project: %w", err)
	}
	return &team, nil
}

// ListProjectMembers retrieves all members of a project via the associated team
func (s *ProjectMemberService) ListProjectMembers(projectID string) ([]models.TeamMember, error) {
	// 1. Get the team associated with the project
	team, err := s.GetProjectTeam(projectID)
	if err != nil {
		return nil, err
	}

	// 2. Get team members using the TeamService
	members, err := s.teamService.ListTeamMembers(team.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}

	return members, nil
}

// AddMemberToProject adds a user to a project by adding them to the associated team
func (s *ProjectMemberService) AddMemberToProject(projectID, userID, inviterID, role string) error {
	// 1. Verify the project exists
	var project models.Item
	if err := repositories.DBS.Postgres.First(&project, "id = ? AND type = ?", projectID, "project").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("project with id %s not found", projectID)
		}
		return fmt.Errorf("failed to fetch project: %w", err)
	}

	// 2. Get the associated team
	team, err := s.GetProjectTeam(projectID)
	if err != nil {
		return err
	}

	// 3. Add the user to the team using TeamService
	return s.teamService.InviteUserToTeam(team.ID, userID, inviterID, role)
}

// RemoveMemberFromProject removes a user from a project by removing them from the associated team
func (s *ProjectMemberService) RemoveMemberFromProject(projectID, userID string) error {
	// 1. Get the team associated with the project
	team, err := s.GetProjectTeam(projectID)
	if err != nil {
		return err
	}

	// 2. Remove the user from the team
	return s.teamService.RemoveUserFromTeam(team.ID, userID)
}

// UpdateProjectMemberRole updates a project member's role by updating their role in the associated team
func (s *ProjectMemberService) UpdateProjectMemberRole(projectID, userID, newRole string) error {
	// 1. Get the team associated with the project
	team, err := s.GetProjectTeam(projectID)
	if err != nil {
		return err
	}

	// 2. Update the user's role in the team
	return s.teamService.UpdateMemberRole(team.ID, userID, newRole)
}

// AcceptProjectInvitation accepts an invitation to join a project team
func (s *ProjectMemberService) AcceptProjectInvitation(membershipID string) error {
	return s.teamService.AcceptTeamInvitation(membershipID)
}

// RejectProjectInvitation rejects an invitation to join a project team
func (s *ProjectMemberService) RejectProjectInvitation(membershipID string) error {
	return s.teamService.RejectTeamInvitation(membershipID)
}

// ListUserProjectInvitations retrieves all pending project invitations for a user
func (s *ProjectMemberService) ListUserProjectInvitations(userID string) ([]models.TeamMember, error) {
	invitations, err := s.teamService.ListUserInvitations(userID)
	if err != nil {
		return nil, err
	}

	// Filter only invitations for teams associated with projects
	var projectInvitations []models.TeamMember
	for _, inv := range invitations {
		if inv.Team != nil && inv.Team.ProjectID != "" {
			projectInvitations = append(projectInvitations, inv)
		}
	}

	return projectInvitations, nil
}

// GetProjectMember retrieves a specific project team member
func (s *ProjectMemberService) GetProjectMember(projectID, userID string) (*models.TeamMember, error) {
	// 1. Get the team associated with the project
	team, err := s.GetProjectTeam(projectID)
	if err != nil {
		return nil, err
	}

	// 2. Find the specific membership
	var membership models.TeamMember
	if err := repositories.DBS.Postgres.Where("team_id = ? AND user_id = ?", team.ID, userID).
		Preload("Profile").
		First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user is not a member of this project")
		}
		return nil, fmt.Errorf("failed to get project member: %w", err)
	}

	return &membership, nil
}

// CheckPermission checks if a user has permission to access the project
func (s *ProjectMemberService) CheckPermission(projectID, userID string) (bool, error) {
	// 1. Get the team associated with the project
	team, err := s.GetProjectTeam(projectID)
	if err != nil {
		return false, err
	}

	// 2. Check if the user is a member of the team
	var membership models.TeamMember
	err = repositories.DBS.Postgres.Where("team_id = ? AND user_id = ?", team.ID, userID).First(&membership).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // User is not a member, but this is not an error
		}
		return false, fmt.Errorf("failed to check project permission: %w", err)
	}

	// User is a member of the project
	return true, nil
}
