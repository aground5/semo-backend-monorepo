package controllers

import (
	"net/http"
	"semo-server/internal/logics"
	"semo-server/internal/middlewares"
	"semo-server/internal/utils"
	"strconv"

	"github.com/labstack/echo/v4"
)

// EntryController handles HTTP requests for entries.
type EntryController struct {
	profileService *logics.ProfileService
	entryService   *logics.EntryService
}

// NewEntryController returns a new instance of EntryController.
func NewEntryController(
	profileService *logics.ProfileService,
	entryService *logics.EntryService,
) *EntryController {
	return &EntryController{
		profileService: profileService,
		entryService:   entryService,
	}
}

// ListEntries handles GET /entries
func (ec *EntryController) ListEntries(c echo.Context) error {
	// 미들웨어에서 이메일 가져오기
	email, err := middlewares.GetEmailFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 서비스 계층에서 프로필 조회
	profile, err := ec.profileService.GetOrCreateProfile(email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	limitStr := c.QueryParam("limit")
	cursor := c.QueryParam("cursor")

	limit := 20 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid limit parameter"})
		}
		limit = parsedLimit
	}

	pagination := utils.CursorPagination{
		Limit:  limit,
		Cursor: cursor,
	}

	result, err := ec.entryService.ListEntriesPaginated(profile.ID, pagination)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}
