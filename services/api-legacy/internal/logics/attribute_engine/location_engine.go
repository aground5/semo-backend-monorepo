package attribute_engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/datatypes"
)

type LocationConfig struct {
	BaseConfig       `json:",inline"`
	DefaultLatitude  float64 `json:"default_latitude"`
	DefaultLongitude float64 `json:"default_longitude"`
	GeocodingEnabled bool    `json:"geocoding_enabled,omitempty"`
	AllowedRegion    string  `json:"allowed_region,omitempty"`
	DistanceUnit     string  `json:"distance_unit,omitempty"`
}

type LocationEngine struct{}

func (le LocationEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg LocationConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid location config format"},
		}
	}
	var errs []string
	if cfg.DefaultLatitude < -90 || cfg.DefaultLatitude > 90 {
		errs = append(errs, "default_latitude must be between -90 and 90")
	}
	if cfg.DefaultLongitude < -180 || cfg.DefaultLongitude > 180 {
		errs = append(errs, "default_longitude must be between -180 and 180")
	}
	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

func (le LocationEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	parts := strings.Split(v, ",")
	if len(parts) != 2 {
		return "", errors.New("location value must be 'lat,long'")
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return "", errors.New("invalid latitude in value")
	}
	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return "", errors.New("invalid longitude in value")
	}
	if lat < -90 || lat > 90 {
		return "", errors.New("latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return "", errors.New("longitude must be between -180 and 180")
	}
	return fmt.Sprintf("%.6f,%.6f", lat, lng), nil
}

func (le LocationEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	return DisplayInfo{
		DisplayValue: value,
		DisplayColor: "#000000",
	}, nil
}

func (le LocationEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[LocationConfig](current, new)
}

func (le LocationEngine) DefaultConfig() datatypes.JSON {
	cfg := LocationConfig{
		DefaultLatitude:  0,
		DefaultLongitude: 0,
		DistanceUnit:     "km",
	}
	return StructToJSON(cfg)
}

func (le LocationEngine) TypeName() string {
	return "location"
}
