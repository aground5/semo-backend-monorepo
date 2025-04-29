package controllers

import (
	"net/http"

	"semo-server/internal/logics"
	"semo-server/internal/middlewares"
	"semo-server/internal/models"
	"semo-server/internal/repositories"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// FileController handles HTTP requests for file upload and download.
type FileController struct {
	fileService           *logics.FileService
	profileService        *logics.ProfileService
	taskPermissionService *logics.TaskPermissionService
}

// NewFileController creates and returns a new FileController instance.
func NewFileController(fileService *logics.FileService, profileService *logics.ProfileService, taskPermissionService *logics.TaskPermissionService) *FileController {
	return &FileController{
		fileService:           fileService,
		profileService:        profileService,
		taskPermissionService: taskPermissionService,
	}
}

// UploadFile handles file upload requests.
// Endpoint: POST /items/:item_id/files
func (fc *FileController) UploadFile(c echo.Context) error {
	// Extract itemID from the URL parameter.
	itemID := c.Param("item_id")
	if itemID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "item_id is required"})
	}

	// Check if the requesting user has write permission on the specified item.
	profile, err := middlewares.GetProfileFromContext(c, fc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	hasPermission, err := fc.taskPermissionService.CheckPermission(itemID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "you do not have the permission"})
	}

	// Parse the multipart form to get the file.
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to get file from request"})
	}
	src, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open uploaded file"})
	}

	// Call FileService.UploadFile to upload the file and create a record.
	uploadedFile, err := fc.fileService.UploadFile(c.Request().Context(), itemID, src, fileHeader)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, uploadedFile)
}

// DownloadFile handles file download requests by generating a presigned URL.
// Endpoint: GET /files/:id
func (fc *FileController) DownloadFile(c echo.Context) error {
	// Extract the file UUID from the URL parameter.
	fileIDStr := c.Param("id")
	if fileIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file id is required"})
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid file id format"})
	}

	// Retrieve the file record from the database.
	var fileRecord models.File
	if err := repositories.DBS.Postgres.First(&fileRecord, "id = ?", fileID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
	}

	// Check if the requesting user has read permission on the associated item.
	profile, err := middlewares.GetProfileFromContext(c, fc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	hasPermission, err := fc.taskPermissionService.CheckPermission(fileRecord.ItemID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "you do not have the permission"})
	}

	// Generate a presigned URL for file download.
	downloadURL, err := fc.fileService.GetDownloadLink(c.Request().Context(), fileID, fileRecord.ItemID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 응답으로 presigned URL만 전송.
	return c.JSON(http.StatusOK, map[string]string{"download_url": downloadURL})
}

// ListFiles handles GET /items/:item_id/files requests.
// 사용자가 해당 item에 대해 read 권한이 있는 경우, DB에서 파일 목록을 조회하여 반환합니다.
func (fc *FileController) ListFiles(c echo.Context) error {
	// URL 파라미터에서 item_id 추출
	itemID := c.Param("item_id")
	if itemID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "item_id is required"})
	}

	// JWT 미들웨어에서 프로필 정보 추출
	profile, err := middlewares.GetProfileFromContext(c, fc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// AuthzService를 이용해 해당 item에 대한 read 권한 확인
	hasPermission, err := fc.taskPermissionService.CheckPermission(itemID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "you do not have the permission"})
	}

	// FileService를 통해 해당 item의 파일 목록 조회
	files, err := fc.fileService.ListFilesByItem(c.Request().Context(), itemID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, files)
}

// DeleteFile handles DELETE /files/:id requests.
// 요청한 사용자가 해당 파일이 속한 item에 대해 write 권한이 있는 경우에만 파일을 삭제합니다.
func (fc *FileController) DeleteFile(c echo.Context) error {
	// URL 파라미터에서 파일 UUID 추출
	fileIDStr := c.Param("id")
	if fileIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file id is required"})
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid file id format"})
	}

	// 파일 레코드를 DB에서 조회
	var fileRecord models.File
	if err := repositories.DBS.Postgres.First(&fileRecord, "id = ?", fileID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
	}

	// JWT 미들웨어로부터 사용자 프로필 추출
	profile, err := middlewares.GetProfileFromContext(c, fc.profileService)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 파일이 속한 item에 대해 write 권한이 있는지 확인
	hasPermission, err := fc.taskPermissionService.CheckPermission(fileRecord.ItemID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "you do not have the permission"})
	}

	// FileService를 통해 S3와 DB에서 파일 삭제
	if err := fc.fileService.DeleteFile(c.Request().Context(), fileID, fileRecord.ItemID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "file deleted successfully"})
}
