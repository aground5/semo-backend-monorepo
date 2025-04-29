package attribute_engine

import (
	"encoding/json"
	"errors"
	"strings"

	"semo-server/main/utils"

	"gorm.io/datatypes"
)

type PhoneConfig struct {
	BaseConfig          `json:",inline"`
	CountryCodeOptional bool `json:"country_code_optional,omitempty"`
}

type PhoneEngine struct{}

func (pe PhoneEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg PhoneConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid phone config format"},
		}
	}
	var errs []string
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

func (pe PhoneEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	// 기본적인 문자 검증; 실제로는 libphonenumber 등의 라이브러리 사용 권장
	for _, r := range v {
		if !(r >= '0' && r <= '9') && r != '+' && r != '-' && r != ' ' && r != '(' && r != ')' {
			return "", errors.New("invalid character in phone number")
		}
	}
	return v, nil
}

func (pe PhoneEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg PhoneConfig
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

func (pe PhoneEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[PhoneConfig](current, new)
}

func (pe PhoneEngine) DefaultConfig() datatypes.JSON {
	cfg := PhoneConfig{
		CountryCodeOptional: true,
		BaseConfig:          BaseConfig{},
	}
	return StructToJSON(cfg)
}

func (pe PhoneEngine) TypeName() string {
	return "phone"
}
