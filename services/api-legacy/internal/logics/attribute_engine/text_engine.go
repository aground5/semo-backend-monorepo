package attribute_engine

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"semo-server/main/utils"

	"golang.org/x/text/unicode/norm"
	"gorm.io/datatypes"
)

// TextConfig는 텍스트 속성에 대한 설정을 정의합니다.
type TextConfig struct {
	BaseConfig       `json:",inline"`
	MaxLength        *int `json:"max_length,omitempty"`
	MinLength        *int `json:"min_length,omitempty"`
	PreserveNewlines bool `json:"preserve_newlines,omitempty"`
}

type TextEngine struct{}

func (te TextEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg TextConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid text config format"},
		}
	}
	var errs []string
	if cfg.MinLength != nil && *cfg.MinLength < 0 {
		errs = append(errs, "min_length must be non-negative")
	}
	if cfg.MaxLength != nil && cfg.MinLength != nil && *cfg.MaxLength < *cfg.MinLength {
		errs = append(errs, "max_length must be greater than or equal to min_length")
	}
	if cfg.DisplayColor != "" {
		if err := utils.ValidateColorString(cfg.DisplayColor); err != nil {
			errs = append(errs, "display_color: "+err.Error())
		}
	}
	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

func (te TextEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	// XSS 방지를 위해 입력값을 sanitization 처리
	sanitized := utils.SanitizeString(value)
	// 유니코드 정규화
	normalized := norm.NFC.String(sanitized)
	if !utf8.ValidString(normalized) {
		return "", errors.New("invalid UTF-8 encoding")
	}
	var cfg TextConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}
	length := len(normalized)
	if cfg.MinLength != nil && length < *cfg.MinLength {
		return "", errors.New("text value is too short")
	}
	if cfg.MaxLength != nil && length > *cfg.MaxLength {
		return "", errors.New("text value is too long")
	}
	if !cfg.PreserveNewlines {
		normalized = strings.ReplaceAll(normalized, "\n", " ")
	}
	return normalized, nil
}

func (te TextEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg TextConfig
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

func (te TextEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[TextConfig](current, new)
}

func (te TextEngine) DefaultConfig() datatypes.JSON {
	cfg := TextConfig{
		BaseConfig:       BaseConfig{},
		MaxLength:        nil,
		MinLength:        nil,
		PreserveNewlines: true,
	}
	return StructToJSON(cfg)
}

func (te TextEngine) TypeName() string {
	return "text"
}
