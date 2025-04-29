package logics

import (
	"fmt"

	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"
)

type EntryService struct {
	cursorManager *utils.CursorManager
}

func NewEntryService(cursorManager *utils.CursorManager) *EntryService {
	return &EntryService{
		cursorManager: cursorManager,
	}
}

type EntryResult struct {
	Entries []models.Entry `json:"entries"`
	utils.PaginationResult
}

// CreateEntry creates a new entry in the database
func (es *EntryService) CreateEntry(entry *models.Entry) (*models.Entry, error) {
	// Generate ID for the entry
	id, err := utils.GenerateUniqueID("EN")
	if err != nil {
		return nil, fmt.Errorf("failed to generate entry ID: %w", err)
	}
	entry.ID = id

	// Create entry in database
	if err := repositories.DBS.Postgres.Create(entry).Error; err != nil {
		return nil, fmt.Errorf("failed to create entry: %w", err)
	}

	// Reload the entry to get all relations
	var result models.Entry
	if err := repositories.DBS.Postgres.
		First(&result, "id = ?", entry.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load created entry: %w", err)
	}

	return &result, nil
}

// ListEntriesPaginated retrieves entries that belong to the specified profile_id,
// with pagination support. Results are sorted by updated_at DESC.
func (es *EntryService) ListEntriesPaginated(profileID string, pagination utils.CursorPagination) (*EntryResult, error) {
	// Set default pagination values
	utils.GetPaginationDefaults(&pagination, 20, 100)

	// Prepare query
	query := repositories.DBS.Postgres.Model(&models.Entry{}).
		Preload("Task").
		Preload("RootTask").
		Where("granted_to = ?", profileID)

	// Apply cursor if provided
	if pagination.Cursor != "" {
		cursorData, err := es.cursorManager.DecodeCursor(pagination.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}

		// Apply cursor condition - get entries older than the cursor or with same timestamp but different ID
		query = query.Where("(created_at < ? OR (created_at = ? AND id < ?))",
			cursorData.Timestamp, cursorData.Timestamp, cursorData.ID)
	}

	// Get one more entry than requested to determine if there are more entries
	query = query.Order("created_at DESC").Limit(pagination.Limit + 1)

	var entries []models.Entry
	if err := query.Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}

	// Check if there are more entries
	hasMore := false
	if len(entries) > pagination.Limit {
		hasMore = true
		entries = entries[:pagination.Limit] // Remove the extra entry
	}

	// Generate next cursor if there are more entries
	nextCursor := ""
	if hasMore && len(entries) > 0 {
		lastEntry := entries[len(entries)-1]
		nextCursor = es.cursorManager.EncodeCursor(lastEntry.CreatedAt, lastEntry.ID)
	}

	return &EntryResult{
		Entries: entries,
		PaginationResult: utils.PaginationResult{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}
