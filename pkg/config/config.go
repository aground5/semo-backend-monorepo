package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 구성 인터페이스
type Config interface {
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetStringMap(key string) map[string]interface{}
	GetStringSlice(key string) []string
	GetAll() map[string]interface{}
	IsSet(key string) bool
}

// 설정 저장소
type config struct {
	data map[string]interface{}
}

// Load 설정 파일 로드
func Load(serviceName string) (Config, error) {
	// 환경 변수에서 환경 가져오기 (기본값: dev)
	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}

	// 현재 디렉토리 또는 SEMO_CONFIG_DIR 환경 변수에서 설정 디렉토리 결정
	configDir := os.Getenv("SEMO_CONFIG_DIR")
	if configDir == "" {
		// 실행 파일 기준 상위 디렉토리에서 configs 디렉토리 찾기
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("실행 파일 경로를 찾을 수 없습니다: %w", err)
		}
		baseDir := filepath.Dir(execPath)
		configDir = filepath.Join(baseDir, "configs")

		// configs 디렉토리가 없으면 상위 디렉토리 시도
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			configDir = filepath.Join(baseDir, "..", "configs")
		}
	}

	// 환경별 설정 파일 경로 생성
	configPath := filepath.Join(configDir, env, fmt.Sprintf("%s.yaml", serviceName))

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("설정 파일을 읽을 수 없습니다 (%s): %w", configPath, err)
	}

	// 환경 변수 치환
	content := os.ExpandEnv(string(data))

	// YAML 파싱
	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("YAML 파싱 오류: %w", err)
	}

	return &config{data: cfg}, nil
}

// Get 설정값 가져오기
func (c *config) Get(key string) interface{} {
	keys := strings.Split(key, ".")
	val := c.findValue(keys, c.data)
	return val
}

// GetString 문자열 설정값 가져오기
func (c *config) GetString(key string) string {
	val := c.Get(key)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", val)
}

// GetInt 정수 설정값 가져오기
func (c *config) GetInt(key string) int {
	val := c.Get(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// GetBool 불리언 설정값 가져오기
func (c *config) GetBool(key string) bool {
	val := c.Get(key)
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// GetStringMap 맵 설정값 가져오기
func (c *config) GetStringMap(key string) map[string]interface{} {
	val := c.Get(key)
	if val == nil {
		return nil
	}
	if m, ok := val.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// GetStringSlice 문자열 배열 설정값 가져오기
func (c *config) GetStringSlice(key string) []string {
	val := c.Get(key)
	if val == nil {
		return nil
	}
	if slice, ok := val.([]interface{}); ok {
		result := make([]string, 0, len(slice))
		for _, v := range slice {
			if s, ok := v.(string); ok {
				result = append(result, s)
			} else {
				result = append(result, fmt.Sprintf("%v", v))
			}
		}
		return result
	}
	return nil
}

// GetAll 모든 설정값 가져오기
func (c *config) GetAll() map[string]interface{} {
	return c.data
}

// IsSet 설정값이 존재하는지 확인
func (c *config) IsSet(key string) bool {
	return c.Get(key) != nil
}

// findValue 중첩된 맵에서 값 찾기
func (c *config) findValue(keys []string, data map[string]interface{}) interface{} {
	if len(keys) == 0 {
		return nil
	}

	key := keys[0]
	val, ok := data[key]
	if !ok {
		return nil
	}

	if len(keys) == 1 {
		return val
	}

	if m, ok := val.(map[string]interface{}); ok {
		return c.findValue(keys[1:], m)
	}

	return nil
}
