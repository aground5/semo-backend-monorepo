// File: /Users/k2zoo/Documents/growingup/ox-hr/main/internal/logics/config_engine/base.go
package config_engine

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
)

// ConfigEngine 인터페이스는 config에 대한 기본 동작을 정의합니다.
type ConfigEngine interface {
	// DefaultConfig는 기본 config JSON을 반환합니다.
	DefaultConfig() datatypes.JSON
	// AllowedKeys는 허용된 키 목록을 반환합니다.
	AllowedKeys() map[string]bool
	// SanitizeConfig는 입력받은 config map에서 허용된 키만 남깁니다.
	SanitizeConfig(configMap map[string]interface{}) map[string]interface{}
	// MergeConfig는 기존 config와 새로운 config를 병합하여 sanitize된 결과를 반환합니다.
	MergeConfig(existing, newConfig datatypes.JSON) (datatypes.JSON, error)
	// GetValue 는 특정 키의 값을 반환합니다.
	GetValue(config datatypes.JSON, key string) (interface{}, error)
}

// BaseConfigEngine는 ConfigEngine 인터페이스의 기본 구현체입니다.
type BaseConfigEngine struct {
	allowedKeys   map[string]bool
	defaultConfig datatypes.JSON
}

// NewBaseConfigEngine는 기본 구현체를 생성합니다.
func NewBaseConfigEngine(allowedKeys map[string]bool, defaultConfig datatypes.JSON) *BaseConfigEngine {
	return &BaseConfigEngine{
		allowedKeys:   allowedKeys,
		defaultConfig: defaultConfig,
	}
}

// DefaultConfig는 기본 설정 JSON을 반환합니다.
func (b *BaseConfigEngine) DefaultConfig() datatypes.JSON {
	return b.defaultConfig
}

// AllowedKeys는 허용된 키 목록을 반환합니다.
func (b *BaseConfigEngine) AllowedKeys() map[string]bool {
	return b.allowedKeys
}

// SanitizeConfig는 입력받은 config map에서 허용된 키만 남깁니다.
func (b *BaseConfigEngine) SanitizeConfig(configMap map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for key, value := range configMap {
		if b.allowedKeys[key] {
			sanitized[key] = value
		}
	}
	return sanitized
}

// MergeConfig는 기존 config와 새로운 config를 병합한 후 sanitize하여 반환합니다.
func (b *BaseConfigEngine) MergeConfig(existing, newConfig datatypes.JSON) (datatypes.JSON, error) {
	var existingMap map[string]interface{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &existingMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal existing config: %w", err)
		}
	} else {
		existingMap = make(map[string]interface{})
	}

	var newMap map[string]interface{}
	if err := json.Unmarshal(newConfig, &newMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal new config: %w", err)
	}

	// 새로운 config의 값(허용된 키만)이 기존 값을 덮어쓰도록 병합합니다.
	for key, value := range newMap {
		if b.allowedKeys[key] {
			existingMap[key] = value
		}
	}

	// 최종적으로 sanitize 처리
	sanitized := b.SanitizeConfig(existingMap)
	bData, err := json.Marshal(sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sanitized config: %w", err)
	}
	return bData, nil
}

// GetValue 는 특정 키의 값을 반환합니다.
func (b *BaseConfigEngine) GetValue(config datatypes.JSON, key string) (interface{}, error) {
	var configMap map[string]interface{}
	if err := json.Unmarshal(config, &configMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	value, ok := configMap[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found", key)
	}
	return value, nil
}
