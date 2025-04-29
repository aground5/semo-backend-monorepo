package attribute_engine

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gorm.io/datatypes"
)

// DropdownOption 각 옵션에 대한 구조
type DropdownOption struct {
	Value        string `json:"value"`
	DisplayColor string `json:"display_color,omitempty"`
}

// DropdownConfig
// - options: 옵션들(각각 value, display_color)
// - display_color: 옵션에 해당되지 않을 경우나 기본 배경색
type DropdownConfig struct {
	BaseConfig `json:",inline"`
	Options    []DropdownOption `json:"options"`
}

type DropdownEngine struct{}

func (de DropdownEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg DropdownConfig
	err := json.Unmarshal(config, &cfg)
	if err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid dropdown config format"},
		}
	}

	var errs []string
	// display_color 검증
	if cfg.DisplayColor != "" {
		if cErr := validateColorString(cfg.DisplayColor); cErr != nil {
			errs = append(errs, "display_color: "+cErr.Error())
		}
	}
	// options 검증
	for i, opt := range cfg.Options {
		if opt.Value == "" {
			errs = append(errs, "options["+strconv.Itoa(i)+"] value is empty")
		}
		if opt.DisplayColor != "" {
			if cErr := validateColorString(opt.DisplayColor); cErr != nil {
				errs = append(errs, "options["+strconv.Itoa(i)+"].display_color: "+cErr.Error())
			}
		}
	}

	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

// ValidateValue => must match one of the options' Value (if non-empty)
func (de DropdownEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}

	var cfg DropdownConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}

	for _, opt := range cfg.Options {
		if opt.Value == v {
			return v, nil
		}
	}
	return "", errors.New("value not found in dropdown options")
}

// GetDisplayInfo => 만약 value가 옵션 중 하나라면 해당 옵션의 color 사용, 없으면 config.display_color
func (de DropdownEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg DropdownConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}
	dispColor := cfg.DisplayColor
	for _, opt := range cfg.Options {
		if opt.Value == value && opt.DisplayColor != "" {
			dispColor = opt.DisplayColor
			break
		}
	}
	return DisplayInfo{
		DisplayValue: value,
		DisplayColor: dispColor,
	}, nil
}

func (de DropdownEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[DropdownConfig](current, new)
}

func (de DropdownEngine) DefaultConfig() datatypes.JSON {
	cfg := DropdownConfig{
		Options: []DropdownOption{},
	}
	return StructToJSON(cfg)
}

func (de DropdownEngine) TypeName() string {
	return "dropdown"
}
