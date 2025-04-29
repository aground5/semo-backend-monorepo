package attribute_engine

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
)

type DateConfig struct {
	BaseConfig    `json:",inline"`
	InvalidColor  string `json:"invalid_color,omitempty"`
	Timezone      string `json:"timezone,omitempty"`
	MinDate       string `json:"min_date,omitempty"`
	MaxDate       string `json:"max_date,omitempty"`
	RelativeDates bool   `json:"relative_dates,omitempty"`
}

type DateEngine struct{}

func (de DateEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg DateConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"날짜 설정 형식이 올바르지 않습니다"},
		}
	}

	var errs []string

	// Timezone 검증
	if cfg.Timezone != "" {
		_, err := time.LoadLocation(cfg.Timezone)
		if err != nil {
			errs = append(errs, "지원하지 않는 시간대입니다: "+cfg.Timezone)
		}
	}

	// 날짜 형식 검증 - 기본 레이아웃
	layout := "2006-01-02"

	// MinDate 검증
	if cfg.MinDate != "" {
		_, err := time.Parse(layout, cfg.MinDate)
		if err != nil {
			errs = append(errs, "최소 날짜 형식이 올바르지 않습니다. YYYY-MM-DD 형식을 사용하세요")
		}
	}

	// MaxDate 검증
	if cfg.MaxDate != "" {
		_, err := time.Parse(layout, cfg.MaxDate)
		if err != nil {
			errs = append(errs, "최대 날짜 형식이 올바르지 않습니다. YYYY-MM-DD 형식을 사용하세요")
		}
	}

	// MinDate, MaxDate 비교 검증
	if cfg.MinDate != "" && cfg.MaxDate != "" {
		minDate, minErr := time.Parse(layout, cfg.MinDate)
		maxDate, maxErr := time.Parse(layout, cfg.MaxDate)

		if minErr == nil && maxErr == nil && minDate.After(maxDate) {
			errs = append(errs, "최소 날짜가 최대 날짜보다 이후입니다")
		}
	}

	// DisplayColor 검증
	if cfg.DisplayColor != "" {
		if err := validateColorString(cfg.DisplayColor); err != nil {
			errs = append(errs, "표시 색상 형식이 올바르지 않습니다: "+err.Error())
		}
	}

	// InvalidColor 검증
	if cfg.InvalidColor != "" {
		if err := validateColorString(cfg.InvalidColor); err != nil {
			errs = append(errs, "오류 색상 형식이 올바르지 않습니다: "+err.Error())
		}
	}

	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

func (de DateEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	var cfg DateConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}

	loc := time.UTC
	if cfg.Timezone != "" {
		if l, err := time.LoadLocation(cfg.Timezone); err == nil {
			loc = l
		} else {
			return "", errors.New("지원하지 않는 시간대입니다: " + cfg.Timezone)
		}
	}
	// 상대 날짜 지원
	if cfg.RelativeDates {
		lowerV := strings.ToLower(v)
		now := time.Now().In(loc)
		if lowerV == "today" {
			return now.Format(time.DateOnly), nil
		}
		if lowerV == "yesterday" {
			return now.AddDate(0, 0, -1).Format(time.DateOnly), nil
		}
		if lowerV == "tomorrow" {
			return now.AddDate(0, 0, 1).Format(time.DateOnly), nil
		}
	}

	// ISO 8601 형식(2025-03-31T15:00:00.000Z) 파싱 시도
	var t time.Time
	var err error
	if strings.Contains(v, "T") {
		t, err = time.Parse(time.RFC3339, v)
		if err != nil {
			// ISO 8601이 아닌 경우 기본 형식으로 파싱 시도
			t, err = time.ParseInLocation(time.DateOnly, v, loc)
			if err != nil {
				return "", errors.New("날짜 형식이 올바르지 않습니다: " + err.Error())
			}
		}
	} else {
		// 기본 날짜 형식(2025-03-31) 파싱 시도
		t, err = time.ParseInLocation(time.DateOnly, v, loc)
		if err != nil {
			return "", errors.New("날짜 형식이 올바르지 않습니다: " + err.Error())
		}
	}

	formatted := t.Format(time.RFC3339)
	if cfg.MinDate != "" {
		if minT, err := time.ParseInLocation(time.DateOnly, cfg.MinDate, loc); err == nil && t.Before(minT) {
			return "", errors.New("지정된 최소 날짜보다 이전 날짜입니다")
		}
	}
	if cfg.MaxDate != "" {
		if maxT, err := time.ParseInLocation(time.DateOnly, cfg.MaxDate, loc); err == nil && t.After(maxT) {
			return "", errors.New("지정된 최대 날짜보다 이후 날짜입니다")
		}
	}
	return formatted, nil
}

func (de DateEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg DateConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}

	color := cfg.DisplayColor
	dispVal := value

	// 빈 값 처리
	if value == "" {
		dispVal = ""
		return DisplayInfo{
			DisplayValue: dispVal,
			DisplayColor: color,
		}, nil
	}

	// 유효성 확인
	_, err := de.ValidateValue(value, config)
	if err != nil && cfg.InvalidColor != "" {
		// 유효하지 않은 값에 대해 InvalidColor 적용
		color = cfg.InvalidColor
	}

	return DisplayInfo{
		DisplayValue: dispVal,
		DisplayColor: color,
	}, nil
}

func (de DateEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[DateConfig](current, new)
}

func (de DateEngine) DefaultConfig() datatypes.JSON {
	cfg := DateConfig{
		BaseConfig:    BaseConfig{},
		RelativeDates: true,
		Timezone:      "UTC",
	}
	return StructToJSON(cfg)
}

func (de DateEngine) TypeName() string {
	return "date"
}
