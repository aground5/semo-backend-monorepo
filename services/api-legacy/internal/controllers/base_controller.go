package controllers

import (
	"semo-server/internal/logics"
	"semo-server/internal/middlewares"
	"semo-server/internal/models"

	"github.com/labstack/echo/v4"
)

// BaseController 모든 컨트롤러에서 공통으로 사용되는 기능 제공
type BaseController struct {
    ProfileService *logics.ProfileService
}

// NewBaseController 새로운 BaseController 인스턴스 생성
func NewBaseController(profileService *logics.ProfileService) BaseController {
    return BaseController{
        ProfileService: profileService,
    }
}

// GetProfileFromEmail 컨텍스트에서 이메일을 가져와 프로필을 조회하거나 생성
func (bc *BaseController) GetProfileFromContext(c echo.Context) (*models.Profile, error) {
    email, err := middlewares.GetEmailFromContext(c)
    if err != nil {
        return nil, err
    }

    profile, err := bc.ProfileService.GetOrCreateProfile(email)
    if err != nil {
        return nil, err
    }
    return profile, nil
}