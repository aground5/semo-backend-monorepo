package controllers

import (
	"net/http"
	"semo-server/internal/logics"
	"semo-server/internal/middlewares"

	"github.com/labstack/echo/v4"
)

// TaskPermissionController 태스크 권한 관련 HTTP 요청 처리
type TaskPermissionController struct {
	taskPermissionService *logics.TaskPermissionService
	profileService        *logics.ProfileService
}

// NewTaskPermissionController 새로운 TaskPermissionController 인스턴스 생성
func NewTaskPermissionController(
	taskPermissionService *logics.TaskPermissionService,
	profileService *logics.ProfileService,
) *TaskPermissionController {
	return &TaskPermissionController{
		taskPermissionService: taskPermissionService,
		profileService:        profileService,
	}
}

// GetPermissions 특정 태스크에 대한 권한이 있는 사용자 목록 조회
// GET /tasks/:id/permissions
func (tpc *TaskPermissionController) GetPermissions(c echo.Context) error {
	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "태스크 ID가 필요합니다"})
	}
	// 미들웨어에서 이메일 가져오기
	email, err := middlewares.GetEmailFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 서비스 계층에서 프로필 조회
	profile, err := tpc.profileService.GetOrCreateProfile(email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 태스크에 대한 읽기 권한 확인
	hasPermission, err := tpc.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 태스크에 대한 읽기 권한이 없습니다"})
	}

	// 권한이 있는 사용자 목록 조회
	profiles, err := tpc.taskPermissionService.ListPermissions(taskID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, profiles)
}

// GrantPermission 특정 사용자에게 태스크 권한 부여
// POST /tasks/:id/permissions
// 요청 본문: {"profile_id": "PR123ABC456DEF"}
func (tpc *TaskPermissionController) GrantPermission(c echo.Context) error {
	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "태스크 ID가 필요합니다"})
	}

	// 요청 본문에서 프로필 ID 추출
	var input struct {
		ProfileID string `json:"profile_id"`
	}
	
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if input.ProfileID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로필 ID가 필요합니다"})
	}
	// 미들웨어에서 이메일 가져오기
	email, err := middlewares.GetEmailFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 서비스 계층에서 프로필 조회
	profile, err := tpc.profileService.GetOrCreateProfile(email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 태스크에 대한 쓰기 권한 확인
	hasPermission, err := tpc.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 태스크에 대한 권한 부여 권한이 없습니다"})
	}

	// 권한 부여
	if err := tpc.taskPermissionService.GrantPermission(taskID, input.ProfileID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "권한이 성공적으로 부여되었습니다"})
}

// RevokePermission 특정 사용자의 태스크 권한 회수
// DELETE /tasks/:id/permissions/:profile_id
func (tpc *TaskPermissionController) RevokePermission(c echo.Context) error {
	taskID := c.Param("id")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "태스크 ID가 필요합니다"})
	}

	profileID := c.Param("profile_id")
	if profileID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로필 ID가 필요합니다"})
	}
	
	// 미들웨어에서 이메일 가져오기
	email, err := middlewares.GetEmailFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 서비스 계층에서 프로필 조회
	profile, err := tpc.profileService.GetOrCreateProfile(email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	if profile.ID == profileID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "자신에게 권한을 회수할 수 없습니다"})
	}

	// 태스크에 대한 쓰기 권한 확인
	hasPermission, err := tpc.taskPermissionService.CheckPermission(taskID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 태스크에 대한 권한 회수 권한이 없습니다"})
	}

	// 권한 회수
	if err := tpc.taskPermissionService.RevokePermission(taskID, profileID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "권한이 성공적으로 회수되었습니다"})
}