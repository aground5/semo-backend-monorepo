package attribute_engine

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gorm.io/datatypes"
)

// MultiSelectOption ...
type MultiSelectOption struct {
	Value        string `json:"value"`
	DisplayColor string `json:"display_color,omitempty"`
}

// MultiSelectConfig
// - options: 여러 개의 option(각각 value, display_color)
// - display_color: 기본 셀 색상
type MultiSelectConfig struct {
	BaseConfig `json:",inline"`
	Options    []MultiSelectOption `json:"options"`
}

type MultiSelectEngine struct{}

func (mse MultiSelectEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg MultiSelectConfig
	err := json.Unmarshal(config, &cfg)
	if err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid multi-select config format"},
		}
	}

	var errs []string
	if cfg.DisplayColor != "" {
		if valErr := validateHex(cfg.DisplayColor); valErr != nil {
			errs = append(errs, "display_color: "+valErr.Error())
		}
	}
	for i, opt := range cfg.Options {
		if opt.Value == "" {
			errs = append(errs, "options["+strconv.Itoa(i)+"] has empty value")
		}
		if opt.DisplayColor != "" {
			if valErr := validateHex(opt.DisplayColor); valErr != nil {
				errs = append(errs, "options["+strconv.Itoa(i)+"].display_color: "+valErr.Error())
			}
		}
	}
	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

// ValidateValue => comma-separated list. Each must be in options.
func (mse MultiSelectEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	var cfg MultiSelectConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}
	optsMap := make(map[string]bool)
	for _, o := range cfg.Options {
		optsMap[o.Value] = true
	}

	selections := strings.Split(v, ",")
	var valid []string
	for _, raw := range selections {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if !optsMap[s] {
			return "", errors.New("value '" + s + "' is not in multi-select options")
		}
		valid = append(valid, s)
	}
	return strings.Join(valid, ","), nil
}

// GetDisplayInfo => 쉼표로 구분된 값들, 각 옵션별로 색상 가능
// 여기서는 간단히 cfg.DisplayColor를 사용. 세부 구현은 필요시 추가
func (mse MultiSelectEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg MultiSelectConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}
	color := cfg.DisplayColor
	return DisplayInfo{
		DisplayValue: value,
		DisplayColor: color,
	}, nil
}

func (mse MultiSelectEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[MultiSelectConfig](current, new)
}

func (mse MultiSelectEngine) DefaultConfig() datatypes.JSON {
	cfg := MultiSelectConfig{
		Options: []MultiSelectOption{},
	}
	return StructToJSON(cfg)
}

func (mse MultiSelectEngine) TypeName() string {
	return "multi-select"
}
