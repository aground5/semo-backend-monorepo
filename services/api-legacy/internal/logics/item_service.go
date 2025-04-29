package logics

import (
	"errors"
	"fmt"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"
	"strings"

	"github.com/shopspring/decimal"

	"semo-server/internal/models"

	"gorm.io/gorm"
)

// ItemResult represents the paginated items result
type ItemResult struct {
	Items []models.Item `json:"items"`
	utils.PaginationResult
}

// ItemService provides business logic for items.
type ItemService struct {
	authzService  *AuthzService
	cursorManager *utils.CursorManager
	db            *gorm.DB
}

// NewItemService creates a new ItemService instance.
func NewItemService(db *gorm.DB, authzService *AuthzService, cursorManager *utils.CursorManager) *ItemService {
	return &ItemService{
		db:            db,
		authzService:  authzService,
		cursorManager: cursorManager,
	}
}

// ListItemsByParentAndTypePaginated retrieves items that belong to the specified parent_id and have the given type,
// with pagination support. Results are sorted by position ASC, then updated_at DESC.
func (is *ItemService) ListItemsByParentAndTypePaginated(userID, parentID, itemType string, pagination utils.CursorPagination) (*ItemResult, error) {
	itemType = strings.ToLower(itemType)

	// Set default pagination values
	utils.GetPaginationDefaults(&pagination, 20, 100)

	// Prepare query
	query := is.db.Model(&models.Item{}).Where("created_by = ?", userID)

	// Apply parent filter
	if parentID != "" {
		query = query.Where("parent_id = ?", parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	// Apply type filter
	query = query.Where("type = ?", itemType)

	// Apply cursor if provided
	if pagination.Cursor != "" {
		cursorData, err := is.cursorManager.DecodeCursor(pagination.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}

		// Apply cursor condition - get items older than the cursor or with same timestamp but different ID
		query = query.Where("(updated_at < ? OR (updated_at = ? AND id < ?))",
			cursorData.Timestamp, cursorData.Timestamp, cursorData.ID)
	}

	// Get one more item than requested to determine if there are more items
	query = query.Order("position ASC").Order("updated_at DESC").Limit(pagination.Limit + 1)

	var items []models.Item
	if err := query.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	// Check if there are more items
	hasMore := false
	if len(items) > pagination.Limit {
		hasMore = true
		items = items[:pagination.Limit] // Remove the extra item
	}

	// Generate next cursor if there are more items
	nextCursor := ""
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		nextCursor = is.cursorManager.EncodeCursor(lastItem.UpdatedAt, lastItem.ID)
	}

	return &ItemResult{
		Items: items,
		PaginationResult: utils.PaginationResult{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

// CreateItemWithOrdering creates a new item with proper ordering using left_item_id.
// If leftItemID is provided, the new item's position is calculated based on the left item within the same group (TaskID).
// If leftItemID is nil, the new item is placed at the end of the group (max position + 1).
func (is *ItemService) CreateItemWithOrdering(input *models.Item, leftItemID *string) (*models.Item, error) {
	// Validate required fields.
	if strings.TrimSpace(input.Type) == "" {
		return nil, fmt.Errorf("item type is required")
	}

	// Generate a unique ID for the new item.
	var prefix string
	switch strings.ToLower(input.Type) {
	case "project":
		prefix = "IP"
	case "task":
		prefix = "IT"
	default:
		return nil, fmt.Errorf("invalid item type: %s", input.Type)
	}
	newID, err := utils.GenerateUniqueID(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to generate item ID: %w", err)
	}
	input.ID = newID

	// Determine the new position.
	if leftItemID != nil {
		newPos, err := recalcNewItemPositionForUpdate(input.ParentID, input.Type, *leftItemID, "")
		if err != nil {
			return nil, err
		}
		input.Position = newPos
	} else {
		// No left_item_id provided → place at the end of the group.
		var maxPos decimal.Decimal
		if input.ParentID == nil || *input.ParentID == "" {
			if err := is.db.
				Model(&models.Item{}).
				Where("parent_id IS NULL AND type = ?", input.Type).
				Select("COALESCE(MAX(position), 0)").
				Scan(&maxPos).Error; err != nil {
				return nil, fmt.Errorf("failed to get last position: %w", err)
			}
		} else {
			if err := is.db.
				Model(&models.Item{}).
				Where("parent_id = ? AND type = ?", *input.ParentID, input.Type).
				Select("COALESCE(MAX(position), 0)").
				Scan(&maxPos).Error; err != nil {
				return nil, fmt.Errorf("failed to get last position: %w", err)
			}
		}
		input.Position = maxPos.Add(decimal.NewFromInt(1))
	}

	input.Color, _ = utils.UniqueIDSvc.GenerateRandomColor()

	// Create the item in the database.
	if err := is.db.Create(&input).Error; err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}
	return input, nil
}

// UpdateItemWithOrdering updates an existing item using left_item_id for ordering.
// The grouping key is TaskID. If left_item_id is provided in the update, the new position is recalculated.
func (is *ItemService) UpdateItemWithOrdering(itemID string, updates models.ItemUpdate, email string) (*models.Item, error) {
	var profile models.Profile
	if err := is.db.First(&profile, "email = ?", email).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	var item models.Item
	if err := is.db.First(&item, "id = ?", itemID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("item with id %s not found", itemID)
		}
		return nil, err
	}

	// Build update map for non-ordering fields.
	updateMap := map[string]interface{}{}
	if updates.Name != nil && *updates.Name != "" {
		updateMap["name"] = *updates.Name
	}
	if updates.Contents != nil {
		updateMap["contents"] = *updates.Contents
	}
	if updates.Objective != nil {
		updateMap["objective"] = *updates.Objective
	}
	if updates.Deliverable != nil {
		updateMap["deliverable"] = *updates.Deliverable
	}
	if updates.Role != nil {
		updateMap["role"] = *updates.Role
	}
	if updates.ParentID != nil {
		updateMap["parent_id"] = *updates.ParentID
	}

	// If left_item_id is provided, recalc the position.
	if updates.LeftItemID != nil && *updates.LeftItemID != item.ID {
		// Determine grouping key: if updates.TaskID is provided, use that; else use current item's TaskID.
		var groupID *string
		if updates.ParentID != nil {
			groupID = updates.ParentID
		} else {
			groupID = item.ParentID
		}
		newPos, err := recalcNewItemPositionForUpdate(groupID, item.Type, *updates.LeftItemID, itemID)
		if err != nil {
			return nil, err
		}
		updateMap["position"] = newPos
	}

	if len(updateMap) == 0 {
		return &item, nil
	}

	if err := is.db.Model(&item).Updates(updateMap).Error; err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	if err := is.db.First(&item, "id = ?", itemID).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve updated item: %w", err)
	}
	return &item, nil
}

// GetItem retrieves an item by its ID.
func (is *ItemService) GetItem(itemID string) (*models.Item, error) {
	var item models.Item
	if err := is.db.First(&item, "id = ?", itemID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("item with id %s not found", itemID)
		}
		return nil, fmt.Errorf("failed to fetch item: %w", err)
	}
	return &item, nil
}

// == private helper function ==

// recalcNewItemPositionForUpdate recalculates the new position for an item
// using the given left_item_id within the same group (items sharing the same parentID).
// excludeItemID is the ID of the item being updated (use empty string if creating a new item).
func recalcNewItemPositionForUpdate(parentID *string, itemType, leftItemID string, excludeItemID string) (decimal.Decimal, error) {
	db := repositories.DBS.Postgres
	return recalcPosition(db, parentID, itemType, leftItemID, excludeItemID)
}

// recalcPosition calculates a new position for an item based on its surrounding items
func recalcPosition(db *gorm.DB, parentID *string, itemType, leftItemID, excludeItemID string) (decimal.Decimal, error) {
	var leftItem models.Item
	if err := db.First(&leftItem, "id = ?", leftItemID).Error; err != nil {
		return decimal.Zero, fmt.Errorf("failed to find left item with id %s: %w", leftItemID, err)
	}

	if parentID == nil && itemType == "task" {
		return decimal.Zero, fmt.Errorf("parentID is nil, so it cannot be used to calculate the new position")
	}

	// Check that leftItem belongs to the same group.
	if parentID == nil {
		if leftItem.ParentID != nil && *leftItem.ParentID != "" {
			return decimal.Zero, fmt.Errorf("left item with id %s does not belong to the root group", leftItemID)
		}
	} else {
		if leftItem.ParentID == nil || *leftItem.ParentID != *parentID {
			return decimal.Zero, fmt.Errorf("left item with id %s does not belong to the specified parent group", leftItemID)
		}
	}

	// Query the right item in the same group (exclude the item being updated).
	var rightItem models.Item
	query := db.
		Where("position > ?", leftItem.Position).
		Where("id <> ?", excludeItemID).
		Order("position ASC")

	// Apply parent filter
	query = applyParentFilter(query, parentID)

	// Add type filter
	query = query.Where("type = ?", itemType)

	err := query.First(&rightItem).Error
	if err != nil {
		// 오른쪽 항목이 없다면, leftItem이 마지막 → new position = leftItem.Position + 1
		return leftItem.Position.Add(decimal.NewFromInt(1)), nil
	}

	newPosition := decimal.Avg(leftItem.Position, rightItem.Position)
	// 최소 간격 체크
	minimumGap, err := decimal.NewFromString(".0000009537")
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to create minimum gap decimal: %w", err)
	}

	if leftItem.Position.Sub(newPosition).Abs().LessThan(minimumGap) {
		// 위치 간격이 너무 작으면 리밸런싱 수행
		return rebalancePositions(db, parentID, itemType, leftItem.ID, excludeItemID)
	}

	return newPosition, nil
}

// rebalancePositions 함수는 특정 그룹 내의 모든 아이템 위치를 재정렬합니다
func rebalancePositions(db *gorm.DB, parentID *string, itemType, leftItemID, excludeItemID string) (decimal.Decimal, error) {
	// 리밸런싱: 해당 그룹에 대해 트랜잭션을 통해 모든 item의 position을 재할당합니다.
	tx := db.Begin()
	if tx.Error != nil {
		return decimal.Zero, fmt.Errorf("failed to begin rebalancing transaction: %w", tx.Error)
	}

	var items []models.Item
	query := tx.Where("type = ?", itemType).Order("position ASC")
	query = applyParentFilter(query, parentID)

	if err := query.Find(&items).Error; err != nil {
		tx.Rollback()
		return decimal.Zero, fmt.Errorf("failed to fetch items for rebalancing: %w", err)
	}

	// 전체 아이템 위치 재조정
	for i, item := range items {
		newPos := decimal.NewFromInt(int64(i + 1))
		if err := tx.Model(&models.Item{}).Where("id = ?", item.ID).Update("position", newPos).Error; err != nil {
			tx.Rollback()
			return decimal.Zero, fmt.Errorf("failed to update position for item id %s: %w", item.ID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return decimal.Zero, fmt.Errorf("failed to commit rebalancing transaction: %w", err)
	}

	// 리밸런싱 후, 재조회
	var leftItem models.Item
	if err := db.First(&leftItem, "id = ?", leftItemID).Error; err != nil {
		return decimal.Zero, fmt.Errorf("failed to re-fetch left item after rebalancing: %w", err)
	}

	// 오른쪽 아이템 재조회
	var rightItemAfter models.Item
	query = db.
		Where("position > ?", leftItem.Position).
		Where("id <> ?", excludeItemID).
		Where("type = ?", itemType).
		Order("position ASC")

	query = applyParentFilter(query, parentID)

	err := query.First(&rightItemAfter).Error
	if err != nil {
		return leftItem.Position.Add(decimal.NewFromInt(1)), nil
	}

	return decimal.Avg(leftItem.Position, rightItemAfter.Position), nil
}

// applyParentFilter는 쿼리에 부모 ID 필터링을 적용합니다
func applyParentFilter(query *gorm.DB, parentID *string) *gorm.DB {
	if parentID == nil {
		return query.Where("parent_id IS NULL")
	}
	return query.Where("parent_id = ?", *parentID)
}
