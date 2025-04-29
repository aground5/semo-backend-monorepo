package logics

import (
	"errors"
	"fmt"
	"net"
	"semo-server/configs"
	"semo-server/internal/logics/config_engine"
	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"
	"strings"

	"gorm.io/gorm"
)

// ProfileService는 프로필 조회, 생성, 수정 등의 기능을 제공합니다.
type ProfileService struct {
	// ProfileConfigEngine을 주입받거나 내부에서 생성할 수 있습니다.
	configEngine config_engine.ConfigEngine
}

// NewProfileService는 ProfileService 인스턴스를 반환합니다.
func NewProfileService(configEngine config_engine.ConfigEngine) *ProfileService {
	return &ProfileService{
		configEngine: configEngine,
	}
}

func (ps *ProfileService) createProfile(email string) (*models.Profile, error) {
	// 새 프로필 생성 (필요한 기본값들을 설정하세요)
	id, err := utils.GenerateUniqueID("P")
	if err != nil {
		return nil, fmt.Errorf("failed to generate team ID: %w", err)
	}

	profile := models.Profile{
		ID:          id,
		Email:       email,
		Name:        strings.Split(email, "@")[0],
		DisplayName: "", // 기본 표시 이름
		Biography:   "",
		Timezone:    "",
		Status:      "active",
		PhotoURL:    "",
		Config:      ps.configEngine.DefaultConfig(),
	}

	if err := repositories.DBS.Postgres.Create(&profile).Error; err != nil {
		return nil, err
	}

	return &profile, nil
}

// GetOrCreateProfile은 email에 해당하는 프로필을 조회하고,
// 없으면 기본값으로 새 프로필을 생성하여 반환합니다.
func (ps *ProfileService) GetOrCreateProfile(email string) (*models.Profile, error) {
	var profile *models.Profile
	result := repositories.DBS.Postgres.First(&profile, "email = ?", email)
	if result.Error != nil {
		// 프로필이 없는 경우
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			createdProfile, err := ps.createProfile(email)
			if err != nil {
				return nil, err
			}
			profile = createdProfile
		}
		// 기타 오류인 경우
		return nil, result.Error
	}
	// 프로필이 이미 존재하면 그대로 반환
	return profile, nil
}

// UpdateProfile은 업데이트 가능한 필드들만 받아 기존 프로필을 갱신합니다.
func (ps *ProfileService) UpdateProfile(userEmail, ip string, updates models.ProfileUpdate) (*models.Profile, error) {
	var profile models.Profile
	if err := repositories.DBS.Postgres.First(&profile, "email = ?", userEmail).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("profile with user email %s not found", userEmail)
		}
		return nil, err
	}

	autoTimezone, err := ps.configEngine.GetValue(profile.Config, "auto_timezone")
	if err != nil {
		return nil, err
	}
	if autoTimezone.(bool) {
		if parsedIp := net.ParseIP(ip); parsedIp != nil && parsedIp.To4() != nil {
			geoIp, err := GetGeoIP(parsedIp.String())
			if err == nil {
				updates.Timezone = &geoIp.TimeZone
			}
		}
	}

	// 만약 업데이트 요청에 config가 포함되었다면, 병합하여 sanitize 처리합니다.
	if updates.Config != nil {
		merged, err := ps.configEngine.MergeConfig(profile.Config, *updates.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to merge profile config: %w", err)
		}
		updates.Config = &merged
	}

	if err := repositories.DBS.Postgres.Model(&profile).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	if err := repositories.DBS.Postgres.First(&profile, "email = ?", userEmail).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve updated profile: %w", err)
	}

	return &profile, nil
}

// GetProfilesByIDs는 여러 ID를 받아서 해당하는 프로필들을 반환합니다.
func (ps *ProfileService) GetProfilesByIDs(ids []string) ([]models.Profile, error) {
	var profiles []models.Profile
	if err := repositories.DBS.Postgres.Where("id IN ?", ids).Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

// CreateInvitedProfile creates a new profile with 'invited' status and sends a registration email
func (ps *ProfileService) CreateInvitedProfile(email string) (*models.Profile, error) {
	// Check if profile already exists
	var existingProfile models.Profile
	result := repositories.DBS.Postgres.First(&existingProfile, "email = ?", email)
	if result.Error == nil {
		return nil, errors.New("profile already exists")
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// Generate unique ID
	id, err := utils.GenerateUniqueID("P")
	if err != nil {
		return nil, fmt.Errorf("failed to generate profile ID: %w", err)
	}

	// Create new profile with invited status
	profile := models.Profile{
		ID:          id,
		Email:       email,
		Name:        strings.Split(email, "@")[0],
		DisplayName: "", // 기본 표시 이름
		Biography:   "",
		Timezone:    "",
		Status:      "invited",
		PhotoURL:    "",
		Config:      ps.configEngine.DefaultConfig(),
	}

	// Save to database
	if err := repositories.DBS.Postgres.Create(&profile).Error; err != nil {
		return nil, err
	}

	// Send registration email
	registrationURL := fmt.Sprintf("%s/register?email=%s", configs.Configs.Service.BaseURL, email)
	err = utils.EmailSvc.SendRegistrationInvitationEmail(
		configs.Configs.Email.SenderEmail,
		email,
		profile.Name,
		registrationURL,
	)

	if err != nil {
		// Log the error but don't fail the profile creation
		fmt.Printf("Failed to send registration email: %v\n", err)
	}

	return &profile, nil
}
