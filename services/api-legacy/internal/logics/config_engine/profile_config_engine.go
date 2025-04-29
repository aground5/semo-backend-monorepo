// File: /Users/k2zoo/Documents/growingup/ox-hr/main/internal/logics/config_engine/profile_config_engine.go
package config_engine

import (
	"encoding/json"
	"gorm.io/datatypes"
)

// ProfileConfigEngine은 프로필 전용 config 엔진입니다.
type ProfileConfigEngine struct {
	*BaseConfigEngine
}

// NewProfileConfigEngine는 ProfileConfigEngine의 인스턴스를 생성합니다.
func NewProfileConfigEngine() *ProfileConfigEngine {
	allowed := map[string]bool{
		"theme":         true,
		"language":      true,
		"auto_timezone": true,
	}

	// 기본 설정을 지정합니다.
	defaultMap := map[string]interface{}{
		"theme":         "light",
		"language":      "en",
		"auto_timezone": true,
	}
	b, err := json.Marshal(defaultMap)
	if err != nil {
		panic("failed to marshal default profile config: " + err.Error())
	}

	return &ProfileConfigEngine{
		BaseConfigEngine: NewBaseConfigEngine(allowed, datatypes.JSON(b)),
	}
}
