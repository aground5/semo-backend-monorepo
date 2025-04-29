package logics

import (
	"errors"
	"fmt"
	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"
	"strings"

	"github.com/shopspring/decimal"

	"gorm.io/gorm"
)

// ProjectResult represents the paginated projects result
type ProjectResult struct {
	Projects []models.Item `json:"projects"`
	utils.PaginationResult
}

// ProjectService provides business logic for projects.
type ProjectService struct {
	cursorManager *utils.CursorManager
}

// NewProjectService creates a new ProjectService instance.
func NewProjectService(cursorManager *utils.CursorManager) *ProjectService {
	return &ProjectService{
		cursorManager: cursorManager,
	}
}

// ListProjectsPaginated retrieves all projects for a user with pagination support.
// Results are sorted by position ASC, then updated_at DESC.
func (ps *ProjectService) ListProjectsPaginated(userID string, pagination utils.CursorPagination) (*ProjectResult, error) {
	// Set default pagination values
	utils.GetPaginationDefaults(&pagination, 20, 100)

	// Prepare query
	query := repositories.DBS.Postgres.Model(&models.Item{}).
		Where("created_by = ? AND type = ? AND parent_id IS NULL", userID, "project")

	// Apply cursor if provided
	if pagination.Cursor != "" {
		cursorData, err := ps.cursorManager.DecodeCursor(pagination.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}

		// Apply cursor condition - get items older than the cursor or with same timestamp but different ID
		query = query.Where("(updated_at < ? OR (updated_at = ? AND id < ?))",
			cursorData.Timestamp, cursorData.Timestamp, cursorData.ID)
	}

	// Get one more item than requested to determine if there are more items
	query = query.Order("position ASC").Order("updated_at DESC").Limit(pagination.Limit + 1)

	var projects []models.Item
	if err := query.Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Check if there are more items
	hasMore := false
	if len(projects) > pagination.Limit {
		hasMore = true
		projects = projects[:pagination.Limit] // Remove the extra item
	}

	// Generate next cursor if there are more items
	nextCursor := ""
	if hasMore && len(projects) > 0 {
		lastProject := projects[len(projects)-1]
		nextCursor = ps.cursorManager.EncodeCursor(lastProject.UpdatedAt, lastProject.ID)
	}

	return &ProjectResult{
		Projects: projects,
		PaginationResult: utils.PaginationResult{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

// CreateProject creates a new project with proper ordering using left_item_id.
// If leftItemID is provided, the new project's position is calculated based on the left project.
// If leftItemID is nil, the new project is placed at the end (max position + 1).
func (ps *ProjectService) CreateProject(input *models.Item, leftItemID *string) (*models.Item, error) {
	// Validate required fields
	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("project name is required")
	}

	// Force type to be "project" and ensure parent_id is nil
	input.Type = "project"
	input.ParentID = nil

	// Generate a unique ID for the new project
	newID, err := utils.GenerateUniqueID("IP")
	if err != nil {
		return nil, fmt.Errorf("failed to generate project ID: %w", err)
	}
	input.ID = newID

	// Determine the new position
	if leftItemID != nil {
		newPos, err := recalcNewItemPositionForUpdate(nil, input.Type, *leftItemID, "")
		if err != nil {
			return nil, err
		}
		input.Position = newPos
	} else {
		// No left_item_id provided → place at the end
		var maxPos decimal.Decimal
		if err := repositories.DBS.Postgres.
			Model(&models.Item{}).
			Where("parent_id IS NULL AND type = ?", input.Type).
			Select("COALESCE(MAX(position), 0)").
			Scan(&maxPos).Error; err != nil {
			return nil, fmt.Errorf("failed to get last position: %w", err)
		}
		input.Position = maxPos.Add(decimal.NewFromInt(1))
	}

	input.Color, _ = utils.UniqueIDSvc.GenerateRandomColor()

	// Start a transaction to create both project and team
	tx := repositories.DBS.Postgres.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Create the project in the database
	if err := tx.Create(&input).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Create a team associated with this project
	teamID, err := utils.GenerateUniqueID("T")
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to generate team ID: %w", err)
	}

	team := models.Team{
		ID:          teamID,
		Name:        input.Name, // 프로젝트 이름과 동일하게 설정
		Description: "Team for project " + input.Name,
		Status:      "active",
		CreatedBy:   input.CreatedBy,
		ProjectID:   input.ID,
	}

	if err := tx.Create(&team).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create team for project: %w", err)
	}

	// Create initial team membership for the project creator
	memberID, err := utils.GenerateUniqueID("M")
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to generate member ID: %w", err)
	}

	membership := models.TeamMember{
		ID:        memberID,
		UserID:    input.CreatedBy,
		TeamID:    teamID,
		Role:      "owner",
		Status:    "active",
		InvitedBy: input.CreatedBy, // Creator invites themself
	}

	if err := tx.Create(&membership).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create team membership: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return input, nil
}

// UpdateProject updates an existing project using left_item_id for ordering.
// If left_item_id is provided in the update, the new position is recalculated.
func (ps *ProjectService) UpdateProject(projectID string, updates models.ItemUpdate, email string) (*models.Item, error) {
	var profile models.Profile
	if err := repositories.DBS.Postgres.First(&profile, "email = ?", email).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	var project models.Item
	if err := repositories.DBS.Postgres.First(&project, "id = ? AND type = ?", projectID, "project").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project with id %s not found", projectID)
		}
		return nil, err
	}

	// Build update map for non-ordering fields
	updateMap := map[string]interface{}{}
	if updates.Name != nil && *updates.Name != "" {
		updateMap["name"] = *updates.Name
	}
	if updates.Contents != nil {
		updateMap["contents"] = *updates.Contents
	}

	// Projects cannot have a parent, so we ignore ParentID if provided

	// If left_item_id is provided, recalc the position
	if updates.LeftItemID != nil && *updates.LeftItemID != project.ID {
		newPos, err := recalcNewItemPositionForUpdate(nil, project.Type, *updates.LeftItemID, projectID)
		if err != nil {
			return nil, err
		}
		updateMap["position"] = newPos
	}

	if len(updateMap) == 0 {
		return &project, nil
	}

	if err := repositories.DBS.Postgres.Model(&project).Updates(updateMap).Error; err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	if err := repositories.DBS.Postgres.First(&project, "id = ?", projectID).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve updated project: %w", err)
	}
	return &project, nil
}

// GetProject retrieves a project by its ID.
func (ps *ProjectService) GetProject(projectID string) (*models.Item, error) {
	var project models.Item
	if err := repositories.DBS.Postgres.First(&project, "id = ? AND type = ?", projectID, "project").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project with id %s not found", projectID)
		}
		return nil, fmt.Errorf("failed to fetch project: %w", err)
	}
	return &project, nil
}

// DeleteProject deletes a project by its ID.
func (ps *ProjectService) DeleteProject(projectID string) error {
	var project models.Item
	if err := repositories.DBS.Postgres.First(&project, "id = ? AND type = ?", projectID, "project").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("project with id %s not found", projectID)
		}
		return fmt.Errorf("failed to fetch project: %w", err)
	}

	// Start a transaction
	tx := repositories.DBS.Postgres.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Delete the project
	if err := tx.Delete(&project).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete project: %w", err)
	}

	// Find and delete the associated team
	var team models.Team
	if err := tx.Where("project_id = ?", projectID).First(&team).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return fmt.Errorf("failed to fetch team for project: %w", err)
		}
		// If team not found, just continue with deletion
	} else {
		// Team found, so delete it
		if err := tx.Delete(&team).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete team: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetProjectChildren retrieves all items that have the specified project as their parent.
func (ps *ProjectService) GetProjectChildren(projectID string, itemType string) ([]models.Item, error) {
	var items []models.Item
	query := repositories.DBS.Postgres.Model(&models.Item{}).
		Where("parent_id = ?", projectID)

	// Filter by type if provided
	if itemType != "" {
		query = query.Where("type = ?", strings.ToLower(itemType))
	}

	if err := query.Order("position ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch project children: %w", err)
	}

	return items, nil
}
