package logics

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"semo-server/internal/logics/attribute_engine"
	"semo-server/internal/models"
)

// AttributeValueService provides business logic for attribute values.
type AttributeValueService struct {
	db *gorm.DB
}

// NewAttributeValueService creates and returns a new instance of AttributeValueService.
func NewAttributeValueService(db *gorm.DB) *AttributeValueService {
	return &AttributeValueService{db: db}
}

// GetAttributeValue retrieves an attribute value for a given attribute and task.
func (avs *AttributeValueService) GetAttributeValue(attributeID int, taskID string) (*models.AttributeValue, error) {
	var attrValue models.AttributeValue
	err := avs.db.
		Where("attribute_id = ? AND task_id = ?", attributeID, taskID).
		First(&attrValue).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("attribute value not found for attribute %d and task %s", attributeID, taskID)
		}
		return nil, fmt.Errorf("failed to fetch attribute value: %w", err)
	}

	return &attrValue, nil
}

// GetAttributeValuesByTask retrieves all attribute values for a specific task.
func (avs *AttributeValueService) GetAttributeValuesByTask(taskID string) ([]models.AttributeValue, error) {
	var values []models.AttributeValue
	if err := avs.db.
		Where("task_id = ?", taskID).
		Preload("Attribute").
		Find(&values).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch attribute values: %w", err)
	}
	return values, nil
}

// EditAttributeValue creates or updates an attribute value for a given attribute and task.
func (avs *AttributeValueService) EditAttributeValue(input *models.AttributeValueUpdate) (*models.AttributeValue, error) {
	if input.AttributeID == 0 {
		return nil, fmt.Errorf("attribute_id is required")
	}
	if input.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	var attr models.Attribute
	if err := avs.db.First(&attr, input.AttributeID).Error; err != nil {
		return nil, fmt.Errorf("failed to load attribute with id %d: %w", input.AttributeID, err)
	}
	attrType := strings.ToLower(attr.Type)

	cleanValue, err := attribute_engine.ValidateAttributeValue(attrType, attr.Config, input.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid attribute value for type '%s': %w", attr.Type, err)
	}

	var existing models.AttributeValue
	err = avs.db.
		Where("attribute_id = ? AND task_id = ?", input.AttributeID, input.TaskID).
		First(&existing).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing attribute value: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		newValue := models.AttributeValue{
			AttributeID: input.AttributeID,
			TaskID:      input.TaskID,
			Value:       cleanValue,
		}
		if err := avs.db.Create(&newValue).Error; err != nil {
			return nil, fmt.Errorf("failed to create attribute value: %w", err)
		}
		return &newValue, nil
	}

	// Update existing value
	if err := avs.db.Model(&existing).Update("value", cleanValue).Error; err != nil {
		return nil, fmt.Errorf("failed to update attribute value: %w", err)
	}

	if err := avs.db.First(&existing, existing.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve updated attribute value: %w", err)
	}
	return &existing, nil
}

// DeleteAttributeValue deletes an attribute value by attribute ID and task ID.
func (avs *AttributeValueService) DeleteAttributeValue(attributeID int, taskID string) error {
	result := avs.db.
		Where("attribute_id = ? AND task_id = ?", attributeID, taskID).
		Delete(&models.AttributeValue{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete attribute value: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("attribute value not found for attribute %d and task %s", attributeID, taskID)
	}

	return nil
}
