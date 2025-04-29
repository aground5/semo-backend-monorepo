package controllers

import (
	"net/http"
	"semo-server/internal/middlewares"
	"strconv"

	"semo-server/internal/logics"
	"semo-server/internal/models"

	"github.com/labstack/echo/v4"
)

// AttributeController handles HTTP requests related to attributes and attribute values.
type AttributeController struct {
	attributeService      *logics.AttributeService
	attributeValueService *logics.AttributeValueService
	profileService        *logics.ProfileService
	taskPermissionService *logics.TaskPermissionService
}

// NewAttributeController returns a new instance of AttributeController.
func NewAttributeController(attributeService *logics.AttributeService, attributeValueService *logics.AttributeValueService, profileService *logics.ProfileService, taskPermissionService *logics.TaskPermissionService) *AttributeController {
	return &AttributeController{
		attributeService:      attributeService,
		attributeValueService: attributeValueService,
		profileService:        profileService,
		taskPermissionService: taskPermissionService,
	}
}

// GetAttributesOfRootTask handles GET /tasks/:id/attributes
// 특정 루트 태스크에 속한 모든 속성을 조회합니다.
func (ac *AttributeController) GetAttributesOfRootTask(c echo.Context) error {
	rootTaskID := c.Param("id")
	if rootTaskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "루트 태스크 ID가 필요합니다"})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, ac.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Check read permission for the root task
	hasPermission, err := ac.taskPermissionService.CheckPermission(rootTaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 태스크의 속성을 볼 권한이 없습니다"})
	}

	// 루트 태스크의 속성 조회
	attributes, err := ac.attributeService.GetAttributeOfRootTask(rootTaskID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, attributes)
}

// GetAttributeValuesByTask handles GET /tasks/:id/attribute-values
// 특정 태스크에 속한 모든 속성 값을 조회합니다.
// 예시 요청:
func (ac *AttributeController) GetAttributeValuesOfTask(c echo.Context) error {
	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "태스크 ID가 필요합니다"})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, ac.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Check read permission for the task
	hasPermission, err := ac.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 태스크의 속성 값을 볼 권한이 없습니다"})
	}

	// 태스크의 속성 값 조회
	attributeValues, err := ac.attributeValueService.GetAttributeValuesByTask(taskID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, attributeValues)
}

// CreateAttribute handles POST /attributes
// 예시 요청:
//
//	{
//	   "root_task_id": "I123ABC456DEF",
//	   "name": "Due Date",
//	   "type": "date",
//	   "config": { "format": "YYYY-MM-DD" },
//	   "left_attr_id": 42   // (선택 사항; 이 값이 있으면 해당 attribute 바로 뒤에 삽입)
//	}
func (ac *AttributeController) CreateAttribute(c echo.Context) error {
	// JSON에 Attribute와 left_attr_id를 함께 받도록 함.
	var input struct {
		models.Attribute
		LeftAttrID *int `json:"left_attr_id"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, ac.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	hasPermission, err := ac.taskPermissionService.CheckPermission(input.RootTaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "you do not have the permission"})
	}

	attr, err := ac.attributeService.CreateAttribute(input.Attribute, input.LeftAttrID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, attr)
}

// UpdateAttribute handles PUT /attributes/:id
// 예시 요청 (partial update):
//
//	{
//	   "name": "New Due Date",
//	   "left_attr_id": 42    // (선택 사항; 이 값이 있으면 해당 attribute 바로 뒤로 재배치)
//	}
func (ac *AttributeController) UpdateAttribute(c echo.Context) error {
	idStr := c.Param("id")
	attributeID, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid attribute id"})
	}

	var updates models.AttributeUpdate
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	attr, err := ac.attributeService.GetAttribute(attributeID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid attribute id"})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, ac.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	hasPermission, err := ac.taskPermissionService.CheckPermission(attr.RootTaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "you do not have the permission"})
	}

	updatedAttr, err := ac.attributeService.UpdateAttribute(attributeID, updates)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, updatedAttr)
}

// EditAttributeValue handles a unified endpoint for creating or updating an attribute value.
// 요청 예시:
//
//	{
//	   "attribute_id": 1,
//	   "task_id": "I123ABC456DEF",
//	   "value": "2023-12-31"
//	}
func (ac *AttributeController) EditAttributeValue(c echo.Context) error {
	var input models.AttributeValueUpdate
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, ac.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	hasPermission, err := ac.taskPermissionService.CheckPermission(input.TaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "you do not have the permission"})
	}

	attrVal, err := ac.attributeValueService.EditAttributeValue(&input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, attrVal)
}

// DeleteAttribute handles DELETE /attributes/:id
// 속성을 삭제합니다. 이 속성에 연결된 모든 속성 값도 함께 삭제됩니다.
func (ac *AttributeController) DeleteAttribute(c echo.Context) error {
	idStr := c.Param("id")
	attributeID, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "유효하지 않은 속성 ID입니다"})
	}

	// 삭제하려는 속성 정보 조회
	attr, err := ac.attributeService.GetAttribute(attributeID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "속성을 찾을 수 없습니다"})
	}

	// JWT 미들웨어에서 프로필 가져오기
	profile, err := middlewares.GetProfileFromContext(c, ac.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 루트 태스크에 대한 권한 확인
	hasPermission, err := ac.taskPermissionService.CheckPermission(attr.RootTaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 속성을 삭제할 권한이 없습니다"})
	}

	// 속성 삭제 실행
	if err := ac.attributeService.DeleteAttribute(attributeID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "속성이 성공적으로 삭제되었습니다"})
}
