package logics

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"semo-server/internal/logics/attribute_engine"
	"semo-server/internal/models"
	"semo-server/internal/repositories"
)

// AttributeValueUpdate is used for partial updates of an attribute value.
type AttributeValueUpdate struct {
	Value *string `json:"value"`
}

// AttributeResult represents the paginated attributes result
type AttributeResult struct {
	Attributes []models.Attribute `json:"attributes"`
}

// AttributeService provides business logic for attributes.
type AttributeService struct {
}

// NewAttributeService creates and returns a new instance of AttributeService.
func NewAttributeService() *AttributeService {
	return &AttributeService{}
}

// GetAttributesByRootTask retrieves all attributes for a specific root task.
func (as *AttributeService) GetAttributesByRootTask(rootTaskID string) ([]models.Attribute, error) {
	var attributes []models.Attribute
	if err := repositories.DBS.Postgres.
		Where("root_task_id = ?", rootTaskID).
		Order("position ASC").
		Find(&attributes).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch attributes: %w", err)
	}
	return attributes, nil
}

// CreateAttribute creates a new attribute for a root task.
// 만약 leftAttrID가 주어지면, 해당 attribute(왼쪽)의 position을 기준으로 새 attribute의 position을 산정합니다.
//   - 동일 루트 태스크 내에서 leftAttrID에 해당하는 attribute 뒤에 오는 attribute가 있다면,
//     두 값의 중간값을 사용하고, 없다면 left attribute의 position + 1을 사용합니다.
//
// 만약 leftAttrID가 nil이면, 루트 태스크 내 최대 position + 1을 사용합니다.
func (as *AttributeService) CreateAttribute(input models.Attribute, leftAttrID *int) (*models.Attribute, error) {
	// 1. 필수 항목 유효성 검사
	if input.RootTaskID == "" {
		return nil, fmt.Errorf("root_task_id is required")
	}
	if !strings.HasPrefix(input.RootTaskID, "I") {
		return nil, fmt.Errorf("root_task_id is not a valid item id (must start with 'I')")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("attribute name is required")
	}
	if input.Type == "" {
		return nil, fmt.Errorf("attribute type is required")
	}
	lowerType := strings.ToLower(input.Type)
	_, engineExists := attribute_engine.GetEngine(lowerType)
	if !engineExists {
		return nil, fmt.Errorf("attribute type '%s' is not allowed", input.Type)
	}

	// 2. Config 기본값 설정 및 검증
	if len(input.Config) == 0 {
		def, err := attribute_engine.DefaultConfigForType(lowerType)
		if err != nil {
			return nil, err
		}
		input.Config = def
	}
	fixed, err := attribute_engine.ValidateAttributeConfig(lowerType, input.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid config for type '%s': %w", input.Type, err)
	}
	input.Config = fixed

	// 3. Position 결정
	if leftAttrID != nil {
		// create 시에는 excludeAttrID는 0 (즉, 제외 대상이 없음)
		newPos, err := as.recalcNewPositionForUpdate(input.RootTaskID, *leftAttrID, 0)
		if err != nil {
			return nil, err
		}
		input.Position = newPos
	} else {
		// leftAttrID가 없으면, 루트 태스크 내 최대 position + 1 사용
		var maxPos decimal.Decimal
		if err := repositories.DBS.Postgres.
			Model(&models.Attribute{}).
			Where("root_task_id = ?", input.RootTaskID).
			Select("COALESCE(MAX(position), 0)").
			Scan(&maxPos).Error; err != nil {
			return nil, fmt.Errorf("failed to get last position: %w", err)
		}
		input.Position = maxPos.Add(decimal.NewFromInt(1))
	}

	// 4. DB에 저장
	if err := repositories.DBS.Postgres.Create(&input).Error; err != nil {
		return nil, fmt.Errorf("failed to create attribute: %w", err)
	}
	return &input, nil
}

// GetAttribute retrieves an attribute by its ID.
func (as *AttributeService) GetAttribute(attributeID int) (*models.Attribute, error) {
	var attr models.Attribute
	if err := repositories.DBS.Postgres.First(&attr, attributeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("attribute with id %d not found", attributeID)
		}
		return nil, fmt.Errorf("failed to get attribute: %w", err)
	}
	return &attr, nil
}

// UpdateAttribute updates an existing attribute by its ID.
// 직접적인 position 업데이트는 허용하지 않고, left_attr_id를 통해서만 재배치가 가능합니다.
func (as *AttributeService) UpdateAttribute(attributeID int, updates models.AttributeUpdate) (*models.Attribute, error) {
	var attr models.Attribute
	if err := repositories.DBS.Postgres.First(&attr, attributeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("attribute with id %d not found", attributeID)
		}
		return nil, fmt.Errorf("failed to get attribute: %w", err)
	}

	// 직접적인 position 업데이트 시도는 금지합니다.
	// (업데이트 구조체에는 position 필드가 없으므로, 오직 left_attr_id로 재배치)
	updateMap := map[string]interface{}{}
	currentType := strings.ToLower(attr.Type)

	// Update name if provided.
	if updates.Name != nil {
		updateMap["name"] = *updates.Name
	}

	// Handle type change and/or config update.
	if updates.Type != nil {
		newType := strings.ToLower(*updates.Type)
		_, engineExists := attribute_engine.GetEngine(newType)
		if !engineExists {
			return nil, fmt.Errorf("attribute type '%s' is not allowed", *updates.Type)
		}
		updateMap["type"] = *updates.Type

		if updates.Config == nil || len(*updates.Config) == 0 {
			def, err := attribute_engine.DefaultConfigForType(newType)
			if err != nil {
				return nil, err
			}
			fixed, err := attribute_engine.ValidateAttributeConfig(newType, def)
			if err != nil {
				return nil, fmt.Errorf("invalid default config for type '%s': %w", *updates.Type, err)
			}
			updateMap["config"] = fixed
		} else {
			def, err := attribute_engine.DefaultConfigForType(newType)
			if err != nil {
				return nil, err
			}
			merged, err := attribute_engine.MergeConfigForType(newType, def, *updates.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to merge config for type '%s': %w", *updates.Type, err)
			}
			fixed, err := attribute_engine.ValidateAttributeConfig(newType, merged)
			if err != nil {
				return nil, fmt.Errorf("invalid config for type '%s': %w", *updates.Type, err)
			}
			updateMap["config"] = fixed
		}
	} else if updates.Config != nil && len(*updates.Config) > 0 {
		merged, err := attribute_engine.MergeConfigForType(currentType, attr.Config, *updates.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to merge config for type '%s': %w", attr.Type, err)
		}
		fixed, err := attribute_engine.ValidateAttributeConfig(currentType, merged)
		if err != nil {
			return nil, fmt.Errorf("invalid config for type '%s': %w", attr.Type, err)
		}
		updateMap["config"] = fixed
	}

	// 만약 클라이언트가 left_attr_id를 전달했다면, 그 기준으로 position 재계산
	if updates.LeftAttrID != nil && *updates.LeftAttrID != attr.ID {
		newPos, err := as.recalcNewPositionForUpdate(attr.RootTaskID, *updates.LeftAttrID, attributeID)
		if err != nil {
			return nil, err
		}
		updateMap["position"] = newPos
	}

	if len(updateMap) == 0 {
		return &attr, nil
	}

	if err := repositories.DBS.Postgres.Model(&attr).Updates(updateMap).Error; err != nil {
		return nil, fmt.Errorf("failed to update attribute: %w", err)
	}

	if err := repositories.DBS.Postgres.First(&attr, attributeID).Error; err != nil {
		return nil, err
	}
	return &attr, nil
}

// DeleteAttribute removes an attribute from the database.
func (as *AttributeService) DeleteAttribute(attributeID int) error {
	var attr models.Attribute
	if err := repositories.DBS.Postgres.First(&attr, attributeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("attribute with id %d not found", attributeID)
		}
		return fmt.Errorf("failed to get attribute: %w", err)
	}

	// Start transaction
	tx := repositories.DBS.Postgres.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin delete transaction: %w", tx.Error)
	}

	// Delete attribute values
	if err := tx.Where("attribute_id = ?", attributeID).Delete(&models.AttributeValue{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete attribute values: %w", err)
	}

	// Delete attribute itself
	if err := tx.Delete(&attr).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete attribute: %w", err)
	}

	return tx.Commit().Error
}

// GetAttributeOfRootTask 특정 루트 태스크에 속한 모든 속성을 조회합니다.
func (as *AttributeService) GetAttributeOfRootTask(rootTaskID string) ([]models.Attribute, error) {
	var attributes []models.Attribute

	// 루트 태스크 ID로 속성 조회
	if err := repositories.DBS.Postgres.Where("root_task_id = ?", rootTaskID).Order("position ASC").Find(&attributes).Error; err != nil {
		return nil, fmt.Errorf("루트 태스크 ID %s의 속성 조회 실패: %w", rootTaskID, err)
	}

	return attributes, nil
}

// == private methods ==

// recalcNewPositionForUpdate is an internal helper that recalculates the new position
// based on a given left_attr_id, excluding the attribute with id=excludeAttrID.
// If excludeAttrID is 0, then no exclusion occurs (used for create).
func (as *AttributeService) recalcNewPositionForUpdate(rootTaskID string, leftAttrID int, excludeAttrID int) (decimal.Decimal, error) {
	var leftAttr models.Attribute
	if err := repositories.DBS.Postgres.First(&leftAttr, leftAttrID).Error; err != nil {
		return decimal.Zero, fmt.Errorf("failed to find left attribute with id %d: %w", leftAttrID, err)
	}
	if leftAttr.RootTaskID != rootTaskID {
		return decimal.Zero, fmt.Errorf("left attribute (id=%d) does not belong to the same root task", leftAttrID)
	}

	rightAttr, err := as.findRightAttribute(rootTaskID, leftAttr.Position, excludeAttrID)
	if err != nil {
		// 오른쪽 속성이 없다면 leftAttr가 마지막이므로, position = leftAttr.Position + 1
		return leftAttr.Position.Add(decimal.NewFromInt(1)), nil
	}

	newPosition := decimal.Avg(leftAttr.Position, rightAttr.Position)
	// 최소 간격 체크 (예: 0.0000009537)
	minimumGap, _ := decimal.NewFromString(".0000009537")
	// 속성 간 간격이 최소 간격보다 작은지 체크 (오른쪽 - 왼쪽 간격)
	if rightAttr.Position.Sub(leftAttr.Position).LessThan(minimumGap) {
		// 간격이 너무 작으면 리밸런싱 수행
		if err := as.rebalanceAttributes(rootTaskID); err != nil {
			return decimal.Zero, err
		}

		// 리밸런싱 후, leftAttr 재조회
		if err := repositories.DBS.Postgres.First(&leftAttr, leftAttr.ID).Error; err != nil {
			return decimal.Zero, fmt.Errorf("failed to re-fetch left attribute after rebalancing: %w", err)
		}

		// 리밸런싱 후, 오른쪽 속성 재조회
		rightAttrAfter, err := as.findRightAttribute(rootTaskID, leftAttr.Position, excludeAttrID)
		if err != nil {
			return leftAttr.Position.Add(decimal.NewFromInt(1)), nil
		}
		return decimal.Avg(leftAttr.Position, rightAttrAfter.Position), nil
	}

	return newPosition, nil
}

// findRightAttribute finds the next attribute to the right of the given position.
func (as *AttributeService) findRightAttribute(rootTaskID string, position decimal.Decimal, excludeAttrID int) (*models.Attribute, error) {
	var rightAttr models.Attribute
	err := repositories.DBS.Postgres.
		Where("root_task_id = ? AND position > ? AND id <> ?", rootTaskID, position, excludeAttrID).
		Order("position ASC").
		First(&rightAttr).Error
	if err != nil {
		return nil, err
	}
	return &rightAttr, nil
}

// rebalanceAttributes rebalances all attributes' positions for a given root task.
// It assigns new sequential positions (1, 2, 3, ...) to all attributes.
func (as *AttributeService) rebalanceAttributes(rootTaskID string) error {
	tx := repositories.DBS.Postgres.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin rebalancing transaction: %w", tx.Error)
	}

	// 속성 목록 조회
	var attrs []models.Attribute
	if err := tx.
		Where("root_task_id = ?", rootTaskID).
		Order("position ASC").
		Find(&attrs).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch attributes for rebalancing: %w", err)
	}

	// 효율적인 일괄 업데이트를 위한 준비
	type UpdateItem struct {
		ID       int
		Position decimal.Decimal
	}
	updates := make([]UpdateItem, len(attrs))

	// 재배열: 각 attribute에 대해 1부터 순차적으로 새로운 position 할당 (예: 1, 2, 3, …)
	for i, a := range attrs {
		updates[i] = UpdateItem{
			ID:       a.ID,
			Position: decimal.NewFromInt(int64(i + 1)),
		}
	}

	// 일괄 업데이트 수행
	for _, item := range updates {
		if err := tx.Model(&models.Attribute{}).
			Where("id = ?", item.ID).
			Update("position", item.Position).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update position for attribute id %d: %w", item.ID, err)
		}
	}

	return tx.Commit().Error
}
