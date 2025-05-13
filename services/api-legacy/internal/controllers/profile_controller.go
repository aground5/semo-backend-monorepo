package controllers

import (
	"net/http"
	"net/mail"
	"semo-server/internal/logics"
	"semo-server/internal/middlewares"
	"semo-server/internal/models"

	"github.com/labstack/echo/v4"
)

// ProfileController는 프로필 관련 HTTP 요청을 처리합니다.
type ProfileController struct {
	BaseController
	searchService  *logics.SearchService
}

// NewProfileController는 ProfileController 인스턴스를 생성합니다.
func NewProfileController(profileService *logics.ProfileService, searchService *logics.SearchService) *ProfileController {
	return &ProfileController{
		BaseController: NewBaseController(profileService),
		searchService:  searchService,
	}
}

// GetProfile은 JWT 미들웨어에서 세팅한 사용자 ID (sub)를 이용해 프로필을 조회합니다.
func (pc *ProfileController) GetProfile(c echo.Context) error {
	userEmail, err := middlewares.GetEmailFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	profile, err = pc.ProfileService.UpdateProfile(userEmail, c.RealIP(), models.ProfileUpdate{})
	return c.JSON(http.StatusOK, profile)
}

// UpdateProfile은 기존 프로필의 데이터를 수정합니다.
// 클라이언트는 업데이트 가능한 필드만 전달하며, JWT의 sub와 비교하여 본인 프로필만 수정하도록 합니다.
func (pc *ProfileController) UpdateProfile(c echo.Context) error {
	var req models.ProfileUpdate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	userEmail, err := middlewares.GetEmailFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	updatedProfile, err := pc.ProfileService.UpdateProfile(userEmail, c.RealIP(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, updatedProfile)
}

// SearchProfiles는 쿼리 파라미터 "email"을 받아, 이메일로 프로필을 검색하여 리스트를 반환합니다.
// URL 예시: GET /profile/search?email=alice@example.com
func (pc *ProfileController) SearchProfile(c echo.Context) error {
	email := c.QueryParam("email")
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email parameter is required"})
	}

	// 이메일 형식 검증
	_, err := mail.ParseAddress(email)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
	}

	result, err := pc.searchService.SearchProfile(email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// GetProfileFromEmail은 컨텍스트에서 이메일을 가져와 프로필을 조회하거나 생성합니다.
// 이 함수는 미들웨어에서 사용되던 로직을 컨트롤러로 이동한 것입니다.
func (pc *ProfileController) GetProfileFromEmail(c echo.Context) (*models.Profile, error) {
	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// CreateInvitedProfile는 이메일을 받아서 초대된 상태의 프로필을 생성하고 회원가입 이메일을 보냅니다.
func (pc *ProfileController) CreateInvitedProfile(c echo.Context) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
	}

	// 이메일 형식 검증
	_, err := mail.ParseAddress(req.Email)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
	}

	profile, err := pc.ProfileService.CreateInvitedProfile(req.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, profile)
}
