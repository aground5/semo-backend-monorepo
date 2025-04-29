package attribute_engine

import (
	"encoding/json"
	"errors"
	"strings"

	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"semo-server/main/utils"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type FileConfig struct {
	BaseConfig       `json:",inline"`
	AllowedFileTypes []string `json:"allowed_file_types,omitempty"`
	MaxSize          int      `json:"max_size,omitempty"`
	MIMEType         string   `json:"mime_type,omitempty"`
	CheckSignature   bool     `json:"check_signature,omitempty"`
	AllowCompressed  bool     `json:"allow_compressed,omitempty"`
	VirusScanEnabled bool     `json:"virus_scan_enabled,omitempty"`
	AllowMultiple    bool     `json:"allow_multiple,omitempty"`
}

type FileEngine struct{}

func (fe FileEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg FileConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid file config format"},
		}
	}
	var errs []string
	if cfg.DisplayColor != "" {
		if err := utils.ValidateColorString(cfg.DisplayColor); err != nil {
			errs = append(errs, "display_color: "+err.Error())
		}
	}
	if cfg.MaxSize < 0 {
		errs = append(errs, "max_size cannot be negative")
	}
	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

func (fe FileEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}

	// 설정 가져오기
	var cfg FileConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}

	// 콤마로 구분된 여러 UUID 처리
	rawIDs := strings.Split(v, ",")
	var dedupRawIDs []string
	seen := make(map[string]bool)

	// 중복 제거
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

	// 다중 파일 허용 여부 확인
	if !cfg.AllowMultiple && len(dedupRawIDs) > 1 {
		return "", errors.New("multiple file IDs provided but allow_multiple=false")
	}

	// 각 UUID 검증
	var validIDs []string
	for _, id := range dedupRawIDs {
		// UUID 형식 확인
		fileID, err := uuid.Parse(id)
		if err != nil {
			return "", errors.New("invalid file uuid format: " + id)
		}

		// 데이터베이스에서 파일 존재 여부 확인
		var count int64
		err = repositories.DBS.Postgres.Model(&models.File{}).
			Where("id = ?", fileID).
			Count(&count).Error
		if err != nil {
			return "", errors.New("failed to check file existence: " + err.Error())
		}

		if count > 0 {
			validIDs = append(validIDs, id)
		}
	}

	if len(validIDs) == 0 {
		return "", errors.New("no valid file UUIDs found")
	}

	return strings.Join(validIDs, ","), nil
}

func (fe FileEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg FileConfig
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

func (fe FileEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[FileConfig](current, new)
}

func (fe FileEngine) DefaultConfig() datatypes.JSON {
	cfg := FileConfig{
		BaseConfig:    BaseConfig{},
		AllowMultiple: true,
	}
	return StructToJSON(cfg)
}

func (fe FileEngine) TypeName() string {
	return "file"
}
