package attribute_engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"semo-server/main/utils"

	"github.com/shopspring/decimal"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gorm.io/datatypes"
)

// NumberConfig는 숫자 속성에 대한 설정을 정의합니다.
type NumberConfig struct {
	BaseConfig           `json:",inline"`
	Min                  *decimal.Decimal `json:"min,omitempty"`
	Max                  *decimal.Decimal `json:"max,omitempty"`
	DecimalPlaces        *int             `json:"decimal_places,omitempty"`
	UseThousandSeparator bool             `json:"use_thousand_separator,omitempty"`
	CurrencyCode         string           `json:"currency_code,omitempty"`
	SymbolPosition       string           `json:"symbol_position,omitempty"` // "before" 또는 "after"
	Unit                 string           `json:"unit,omitempty"`
	ScaleFactor          *decimal.Decimal `json:"scale_factor,omitempty"`
}

type NumberEngine struct{}

func (ne NumberEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg NumberConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid config format"},
		}
	}
	var errorsList []string
	if cfg.Min != nil && cfg.Max != nil && cfg.Min.GreaterThan(*cfg.Max) {
		errorsList = append(errorsList, "min cannot be greater than max")
	}
	if cfg.DecimalPlaces != nil && *cfg.DecimalPlaces < 0 {
		errorsList = append(errorsList, "decimal_places must be non-negative")
	}
	if cfg.DisplayColor != "" {
		if err := utils.ValidateColorString(cfg.DisplayColor); err != nil {
			errorsList = append(errorsList, "display_color: "+err.Error())
		}
	}
	if cfg.SymbolPosition != "" && cfg.SymbolPosition != "before" && cfg.SymbolPosition != "after" {
		errorsList = append(errorsList, "symbol_position must be 'before' or 'after'")
	}
	fixed := StructToJSON(cfg)
	return ConfigValidationResult{
		IsValid: len(errorsList) == 0,
		Errors:  errorsList,
		Fixed:   fixed,
	}
}

func (ne NumberEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	d, err := decimal.NewFromString(v)
	if err != nil {
		return "", fmt.Errorf("invalid number format: %v", err)
	}
	var cfg NumberConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}
	if cfg.Min != nil && d.LessThan(*cfg.Min) {
		return "", fmt.Errorf("value is less than minimum (%v)", cfg.Min)
	}
	if cfg.Max != nil && d.GreaterThan(*cfg.Max) {
		return "", fmt.Errorf("value is greater than maximum (%v)", cfg.Max)
	}
	// 적용할 scale_factor가 있다면 곱셈 적용
	if cfg.ScaleFactor != nil {
		d = d.Mul(*cfg.ScaleFactor)
	}
	dp := 2
	if cfg.DecimalPlaces != nil {
		dp = *cfg.DecimalPlaces
	}
	d = d.Round(int32(dp))
	formatted := d.StringFixed(int32(dp))
	if cfg.UseThousandSeparator {
		p := message.NewPrinter(language.English)
		formatted = p.Sprintf("%s", formatted)
	}
	if cfg.Unit != "" {
		formatted += " " + cfg.Unit
	}
	if cfg.CurrencyCode != "" {
		symbol := cfg.CurrencyCode
		if cfg.SymbolPosition == "before" {
			formatted = symbol + " " + formatted
		} else {
			formatted = formatted + " " + symbol
		}
	}
	return formatted, nil
}

func (ne NumberEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg NumberConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}
	color := "#000000"
	if cfg.DisplayColor != "" {
		color = cfg.DisplayColor
	}
	extra := map[string]any{}
	d, err := decimal.NewFromString(value)
	if err == nil {
		extra["is_negative"] = d.LessThan(decimal.Zero)
		extra["magnitude"] = d.Abs().String()
	}
	return DisplayInfo{
		DisplayValue: value,
		DisplayColor: color,
		ExtraInfo:    extra,
	}, nil
}

func (ne NumberEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[NumberConfig](current, new)
}

func (ne NumberEngine) DefaultConfig() datatypes.JSON {
	dp := 2
	cfg := NumberConfig{
		BaseConfig:           BaseConfig{},
		DecimalPlaces:        &dp,
		UseThousandSeparator: true,
	}
	return StructToJSON(cfg)
}

func (ne NumberEngine) TypeName() string {
	return "number"
}
