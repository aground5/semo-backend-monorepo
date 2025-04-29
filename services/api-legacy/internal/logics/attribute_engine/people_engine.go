package attribute_engine

import (
	"encoding/json"
	"errors"
	"strings"

	"semo-server/internal/models"
	"semo-server/internal/repositories"

	"gorm.io/datatypes"
)

type PeopleConfig struct {
	BaseConfig    `json:",inline"`
	AllowMultiple bool `json:"allow_multiple,omitempty"`
}

type PeopleEngine struct{}

func (pe PeopleEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg PeopleConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid people config format"},
		}
	}
	var errs []string
	if cfg.DisplayColor != "" {
		if len(cfg.DisplayColor) != 7 || !strings.HasPrefix(cfg.DisplayColor, "#") {
			errs = append(errs, "display_color must be # followed by 6 hex digits")
		}
	}
	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

func (pe PeopleEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	var cfg PeopleConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}
	rawIDs := strings.Split(v, ",")
	var dedupRawIDs []string
	seen := make(map[string]bool)
	for _, id := range rawIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if !seen[id] {
			seen[id] = true
			dedupRawIDs = append(dedupRawIDs, id)
		}
	}
	var validIDs []string
	for _, id := range dedupRawIDs {
		var count int64
		err := repositories.DBS.Postgres.Model(&models.Profile{}).
			Where("id = ?", id).
			Count(&count).Error
		if err != nil {
			return "", errors.New("failed to check profile existence: " + err.Error())
		}
		if count == 0 {
			err = repositories.DBS.Postgres.Model(&models.Team{}).
				Where("id = ?", id).
				Count(&count).Error
			if err != nil {
				return "", errors.New("failed to check team existence: " + err.Error())
			}
		}
		if count > 0 {
			validIDs = append(validIDs, id)
		}
	}
	if len(validIDs) == 0 {
		return "", errors.New("no valid profile or team IDs found")
	}
	if !cfg.AllowMultiple && len(validIDs) > 1 {
		return "", errors.New("multiple IDs provided but allow_multiple=false")
	}
	return strings.Join(validIDs, ","), nil
}

func (pe PeopleEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg PeopleConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}
	color := "#000000"
	if cfg.DisplayColor != "" {
		color = cfg.DisplayColor
	}
	return DisplayInfo{
		DisplayValue: value,
		DisplayColor: color,
	}, nil
}

func (pe PeopleEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[PeopleConfig](current, new)
}

func (pe PeopleEngine) DefaultConfig() datatypes.JSON {
	cfg := PeopleConfig{
		AllowMultiple: true,
		BaseConfig:    BaseConfig{},
	}
	return StructToJSON(cfg)
}

func (pe PeopleEngine) TypeName() string {
	return "people"
}
