package attribute_engine

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gorm.io/datatypes"
)

type SelectOption struct {
	Value        string `json:"value"`
	DisplayColor string `json:"display_color,omitempty"`
}

// SelectConfig
// - options: 단일 선택 옵션(각각 value, display_color)
// - display_color: 기본 셀 색상
type SelectConfig struct {
	BaseConfig `json:",inline"`
	Options    []SelectOption `json:"options"`
}

type SelectEngine struct{}

func (se SelectEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg SelectConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid select config format"},
		}
	}

	var errs []string
	if cfg.DisplayColor != "" {
		if errVal := validateColorString(cfg.DisplayColor); errVal != nil {
			errs = append(errs, "display_color: "+errVal.Error())
		}
	}
	for i, opt := range cfg.Options {
		if opt.Value == "" {
			errs = append(errs, "options["+strconv.Itoa(i)+"].value is empty")
		}
		if opt.DisplayColor != "" {
			if errVal := validateColorString(opt.DisplayColor); errVal != nil {
				errs = append(errs, "options["+strconv.Itoa(i)+"].display_color: "+errVal.Error())
			}
		}
	}

	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

func (se SelectEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	var cfg SelectConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}
	for _, opt := range cfg.Options {
		if opt.Value == v {
			return v, nil
		}
	}
	return "", errors.New("value not found in select options")
}

// GetDisplayInfo => 만약 value가 옵션 중 하나라면 해당 옵션 color 사용, 없으면 config.display_color
func (se SelectEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg SelectConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}
	color := cfg.DisplayColor
	for _, opt := range cfg.Options {
		if opt.Value == value && opt.DisplayColor != "" {
			color = opt.DisplayColor
			break
		}
	}
	return DisplayInfo{
		DisplayValue: value,
		DisplayColor: color,
	}, nil
}

func (se SelectEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[SelectConfig](current, new)
}

func (se SelectEngine) DefaultConfig() datatypes.JSON {
	cfg := SelectConfig{
		Options: []SelectOption{},
	}
	return StructToJSON(cfg)
}

func (se SelectEngine) TypeName() string {
	return "select"
}
