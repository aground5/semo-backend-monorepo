package controllers

import (
	"net/http"
	"semo-server/internal/logics"

	"github.com/labstack/echo/v4"
)

// ShareController 공유 관련 HTTP 요청 처리
type ShareController struct {
	taskPermissionService *logics.TaskPermissionService
}

// NewShareController 새로운 ShareController 인스턴스 생성
func NewShareController(
	taskPermissionService *logics.TaskPermissionService,
) *ShareController {
	return &ShareController{
		taskPermissionService: taskPermissionService,
	}
}

// GetSharedTaskFromUUID UUID를 통해 공유된 태스크를 조회하는 함수
// GET /share/:uuid
func (sc *ShareController) GetSharedTaskFromUUID(c echo.Context) error {
	// URL 파라미터에서 UUID 추출
	uuid := c.Param("uuid")
	if uuid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "UUID가 필요합니다"})
	}

	// 서비스를 통해 태스크와 하위 태스크 조회
	task, childTasks, err := sc.taskPermissionService.GetTaskAndChildrenByShareUUID(uuid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"task":        task,
		"child_tasks": childTasks,
	})
}
