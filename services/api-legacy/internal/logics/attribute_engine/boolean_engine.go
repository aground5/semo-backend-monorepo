package attribute_engine

import (
	"encoding/json"
	"errors"
	"strings"

	"semo-server/main/utils"

	"gorm.io/datatypes"
)

type BooleanConfig struct {
	BaseConfig        `json:",inline"`
	TrueDisplayColor  string `json:"true_display_color,omitempty"`
	FalseDisplayColor string `json:"false_display_color,omitempty"`
	TrueLabel         string `json:"true_label,omitempty"`
	FalseLabel        string `json:"false_label,omitempty"`
	Nullable          bool   `json:"nullable,omitempty"`
	ToggleStyle       string `json:"toggle_style,omitempty"`
}

type BooleanEngine struct{}

func (be BooleanEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg BooleanConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid boolean config format"},
		}
	}
	var errs []string
	for _, c := range []struct {
		fieldName string
		value     string
	}{
		{"display_color", cfg.DisplayColor},
		{"true_display_color", cfg.TrueDisplayColor},
		{"false_display_color", cfg.FalseDisplayColor},
	} {
		if c.value != "" {
			if errCol := utils.ValidateColorString(c.value); errCol != nil {
				errs = append(errs, c.fieldName+": "+errCol.Error())
			}
		}
	}
	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

func (be BooleanEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	lowerVal := strings.ToLower(strings.TrimSpace(value))
	if lowerVal == "" {
		var cfg BooleanConfig
		if err := json.Unmarshal(config, &cfg); err == nil && cfg.Nullable {
			return "", nil
		}
		return "", errors.New("value is empty and boolean is not nullable")
	}
	switch lowerVal {
	case "true", "1", "yes", "y":
		return "true", nil
	case "false", "0", "no", "n":
		return "false", nil
	default:
		return "", errors.New("value must be a boolean string (true/false/1/0/yes/no)")
	}
}

func (be BooleanEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg BooleanConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}
	var dispVal string
	var color string = "#000000"
	if value == "true" {
		if cfg.TrueLabel != "" {
			dispVal = cfg.TrueLabel
		} else {
			dispVal = "Yes"
		}
		if cfg.TrueDisplayColor != "" {
			color = cfg.TrueDisplayColor
		} else if cfg.DisplayColor != "" {
			color = cfg.DisplayColor
		}
	} else if value == "false" {
		if cfg.FalseLabel != "" {
			dispVal = cfg.FalseLabel
		} else {
			dispVal = "No"
		}
		if cfg.FalseDisplayColor != "" {
			color = cfg.FalseDisplayColor
		} else if cfg.DisplayColor != "" {
			color = cfg.DisplayColor
		}
	} else {
		dispVal = ""
		if cfg.DisplayColor != "" {
			color = cfg.DisplayColor
		}
	}
	return DisplayInfo{
		DisplayValue: dispVal,
		DisplayColor: color,
	}, nil
}

func (be BooleanEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[BooleanConfig](current, new)
}

func (be BooleanEngine) DefaultConfig() datatypes.JSON {
	cfg := BooleanConfig{
		BaseConfig: BaseConfig{},
		TrueLabel:  "Yes",
		FalseLabel: "No",
		Nullable:   false,
	}
	return StructToJSON(cfg)
}

func (be BooleanEngine) TypeName() string {
	return "boolean"
}
