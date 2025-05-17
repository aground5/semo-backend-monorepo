package controllers

import (
	"net/http"

	"semo-server/internal/logics"
	"semo-server/internal/middlewares"
	"semo-server/internal/models"
	"semo-server/internal/utils"

	"github.com/labstack/echo/v4"
)

// TaskController handles HTTP requests for tasks.
type TaskController struct {
	taskService           *logics.TaskService
	profileService        *logics.ProfileService
	taskPermissionService *logics.TaskPermissionService
	projectMemberService  *logics.ProjectMemberService
	entryService          *logics.EntryService
	attributeService      *logics.AttributeService
	attributeValueService *logics.AttributeValueService
}

// NewTaskController returns a new instance of TaskAPIController.
func NewTaskController(
	taskService *logics.TaskService,
	profileService *logics.ProfileService,
	taskPermissionService *logics.TaskPermissionService,
	projectMemberService *logics.ProjectMemberService,
	entryService *logics.EntryService,
	attributeService *logics.AttributeService,
	attributeValueService *logics.AttributeValueService,
) *TaskController {
	return &TaskController{
		taskService:           taskService,
		profileService:        profileService,
		taskPermissionService: taskPermissionService,
		projectMemberService:  projectMemberService,
		entryService:          entryService,
		attributeService:      attributeService,
		attributeValueService: attributeValueService,
	}
}

// GetTask handles GET /tasks/:id requests.
func (tc *TaskController) GetTask(c echo.Context) error {
	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "task id is required"})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, tc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Check read permission for the task.
	hasPermission, err := tc.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have the permission"})
	}

	// Fetch the task.
	task, err := tc.taskService.GetTask(taskID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, task)
}

// CreateTask handles POST /tasks
// 요청 예시:
//
//	{
//	  "parent_id": "IP123ABC456DEF",    // 부모 항목 ID (옵션)
//	  "name": "New Task",
//	  "contents": "내용...",
//	  "left_item_id": "ID987XYZ654ABC"   // (선택 사항; 이 값이 있으면 해당 item 바로 뒤에 삽입)
//	}
func (tc *TaskController) CreateTask(c echo.Context) error {
	// left_item_id를 함께 받기 위해 별도 구조체 정의
	var input struct {
		models.Item
		LeftItemID *string `json:"left_item_id"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// 이름이 없는 경우 BadRequest 반환
	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, tc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Set the creator ID
	input.Item.CreatedBy = profile.ID

	// Check permission for the parent if specified
	if input.ParentID != nil && *input.ParentID != "" {
		hasPermission, err := tc.taskPermissionService.CheckPermission(*input.ParentID, profile.ID)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
		if !hasPermission {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have the permission"})
		}
	}

	task, err := tc.taskService.CreateTask(&input.Item, input.LeftItemID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, task)
}

// UpdateTask handles PUT /tasks/:id
// 요청 예시 (부분 업데이트):
//
//		{
//		  "name": "Updated Task Name",
//		  "left_item_id": "ID987XYZ654ABC",   // (선택 사항; 이 값이 있으면 해당 item 바로 뒤로 재배치)
//		  "parent_id": "IP123ABC456DEF",       // (필요하다면 그룹 변경)
//	   "granted_to": "PR123456789"          // (태스크 할당 업데이트)
//		}
func (tc *TaskController) UpdateTask(c echo.Context) error {
	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "task id is required"})
	}

	// Parse update data
	var updates models.ItemUpdate
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, tc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Check write permission for the task.
	hasPermission, err := tc.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have the permission"})
	}

	// If parent_id is being changed, check write permission for new parent
	if updates.ParentID != nil && *updates.ParentID != "" {
		if len(*updates.ParentID) < 2 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid parent id"})
		}

		// If parent_id is a project, check project permission
		if (*updates.ParentID)[:2] == "IP" {
			hasPermission, err := tc.projectMemberService.CheckPermission(*updates.ParentID, profile.ID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
			}
			if !hasPermission {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have the permission to move to the specified parent"})
			}
		} else {
			// If parent_id is a task, check task permission
			hasPermission, err := tc.taskPermissionService.CheckPermission(*updates.ParentID, profile.ID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
			}
			if !hasPermission {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have the permission to move to the specified parent"})
			}
		}
	}

	// Update the task
	task, err := tc.taskService.UpdateTask(taskID, updates)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, task)
}

// DeleteTask handles DELETE /tasks/:id
func (tc *TaskController) DeleteTask(c echo.Context) error {
	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "task id is required"})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, tc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Check write permission for the task.
	hasPermission, err := tc.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have the permission"})
	}

	// Delete the task
	if err := tc.taskService.DeleteTask(taskID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// GetChildTasks handles GET /tasks/:id/children
// 특정 task를 부모로 하는 모든 자식 task를 조회합니다. (단 depth는 1단계만 내려갑니다.)
func (tc *TaskController) GetChildTasks(c echo.Context) error {
	parentID := c.Param("id")
	if parentID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "부모 task id가 필요합니다"})
	}

	// Retrieve profile from JWT middleware
	profile, err := middlewares.GetProfileFromContext(c, tc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Check read permission for the parent task
	hasPermission, err := tc.taskPermissionService.CheckPermission(parentID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 태스크를 볼 권한이 없습니다"})
	}

	// 커서 기반 페이지네이션 파라미터 추출
	pagination := utils.ExtractCursorPaginationFromContext(c)
	utils.GetPaginationDefaults(&pagination, 20, 100) // 기본값 설정: 기본 20개, 최대 100개

	// 자식 태스크 조회
	result, err := tc.taskService.GetChildTasks(parentID, pagination)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// InviteNewParticipant handles POST /tasks/:id/invite
func (tc *TaskController) InviteNewParticipant(c echo.Context) error {
	profile, err := middlewares.GetProfileFromContext(c, tc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "task id is required"})
	}

	task, err := tc.taskService.GetTask(taskID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	hasPermission, err := tc.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have the permission"})
	}

	// request body 파싱
	var invite models.ItemInvite
	if err := c.Bind(&invite); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if invite.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
	}

	// 초대 프로필 생성 및 메일 발송
	invitedProfile, err := tc.profileService.CreateInvitedProfile(invite.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	rootTaskID := tc.taskService.FindRootTaskID(task.ID)

	// Entry 생성
	entry := &models.Entry{
		Name:       task.Name,
		TaskID:     task.ID,
		RootTaskID: rootTaskID,
		CreatedBy:  profile.ID,
		GrantedTo:  invitedProfile.ID,
	}
	if _, err := tc.entryService.CreateEntry(entry); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Attribute 생성
	attribute := models.Attribute{
		RootTaskID: rootTaskID,
		Name:       "task_participant",
		Type:       "people",
	}

	attr, err := tc.attributeService.CreateAttribute(attribute, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// AttributeValue 생성
	attrValueUpdate := &models.AttributeValueUpdate{
		AttributeID: attr.ID,
		TaskID:      task.ID,
		Value:       invitedProfile.ID,
	}

	if _, err := tc.attributeValueService.EditAttributeValue(attrValueUpdate); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "초대 메일 발송 완료"})
}
