package controllers

import (
	"net/http"
	"semo-server/internal/logics"
	"semo-server/internal/models"
	"semo-server/internal/repositories"

	"github.com/labstack/echo/v4"
)

// SharePermissionController 공유 권한 관련 HTTP 요청 처리
type SharePermissionController struct {
	BaseController
	taskPermissionService *logics.TaskPermissionService
	shareService          *logics.ShareService
}

// NewSharePermissionController 새로운 SharePermissionController 인스턴스 생성
func NewSharePermissionController(
	taskPermissionService *logics.TaskPermissionService,
	profileService *logics.ProfileService,
	shareService *logics.ShareService,
) *SharePermissionController {
	return &SharePermissionController{
		BaseController: NewBaseController(profileService),
		taskPermissionService: taskPermissionService,
		shareService:          shareService,
	}
}

// GetShareUUID UUID를 통해 권한을 확인하는 함수
// GET /tasks/:id/share
func (spc *SharePermissionController) GetShareUUID(c echo.Context) error {
	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "태스크 ID가 필요합니다"})
	}
	profile, err := spc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 태스크에 대한 권한 확인
	hasPermission, err := spc.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 태스크에 대한 권한이 없습니다"})
	}

	// UUID 조회
	uuid, err := spc.shareService.GetShareUUID(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"message": "UUID 조회 실패", "uuid": "", "exists": false})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "UUID 조회 성공", "uuid": uuid, "exists": true})
}

// GrantPermissionFromUUID 프로젝트 ID를 받아 UUID를 생성하고 권한을 부여하는 함수
// POST /share
func (spc *SharePermissionController) GrantPermissionFromUUID(c echo.Context) error {
	// 요청 본문에서 프로젝트 ID 추출
	var input struct {
		TaskID string `json:"task_id"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if input.TaskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	profile, err := spc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트에 대한 권한 확인
	hasPermission, err := spc.taskPermissionService.CheckPermission(input.TaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 권한 부여 권한이 없습니다"})
	}

	// 이미 생성된 UUID가 있는지 확인
	exists, err := spc.taskPermissionService.CheckShareExists(input.TaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if exists {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "이미 생성된 UUID가 있습니다"})
	}

	// 새로운 UUID 생성 및 권한 부여
	uuid, err := spc.taskPermissionService.GrantPermissionWithUUID(input.TaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "권한이 성공적으로 부여되었습니다",
		"uuid":    uuid,
	})
}

// RevokePermissionFromUUID UUID를 통해 권한을 회수하는 함수
// DELETE /share/:uuid
func (spc *SharePermissionController) RevokePermissionFromUUID(c echo.Context) error {
	// URL 파라미터에서 UUID 추출
	uuid := c.Param("uuid")
	if uuid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "UUID가 필요합니다"})
	}
	profile, err := spc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// UUID로 공유 조회하여 태스크 ID 확인
	var share models.Share
	if err := repositories.DBS.Postgres.First(&share, "id = ?", uuid).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "권한을 찾을 수 없습니다"})
	}

	// 태스크에 대한 권한 확인
	hasPermission, err := spc.taskPermissionService.CheckPermission(share.RootTaskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 태스크에 대한 권한 회수 권한이 없습니다"})
	}

	// 권한 회수
	if err := spc.taskPermissionService.RevokePermissionWithUUID(uuid); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "권한이 성공적으로 회수되었습니다"})
}
