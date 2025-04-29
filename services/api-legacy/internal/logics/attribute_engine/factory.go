package attribute_engine

import (
	"fmt"
	"strings"

	"gorm.io/datatypes"
)

var engines map[string]AttributeEngine

func init() {
	engines = make(map[string]AttributeEngine)
	RegisterEngine(TextEngine{})
	RegisterEngine(NumberEngine{})
	RegisterEngine(DateEngine{})
	RegisterEngine(BooleanEngine{})
	RegisterEngine(SelectEngine{})
	RegisterEngine(MultiSelectEngine{})
	RegisterEngine(DropdownEngine{})
	RegisterEngine(URLEngine{})
	RegisterEngine(EmailEngine{})
	RegisterEngine(FileEngine{})
	RegisterEngine(LocationEngine{})
	RegisterEngine(PhoneEngine{})
	RegisterEngine(PeopleEngine{})
}

// RegisterEngine은 새로운 속성 엔진을 등록합니다.
func RegisterEngine(engine AttributeEngine) {
	engines[engine.TypeName()] = engine
}

// GetEngine은 주어진 속성 타입 이름에 해당하는 엔진을 반환합니다.
func GetEngine(typeName string) (AttributeEngine, bool) {
	engine, ok := engines[typeName]
	if !ok {
		engine, ok = engines[lowercase(typeName)]
	}
	return engine, ok
}

// ValidateAttributeConfig은 주어진 속성 타입의 config를 해당 엔진을 이용해 검증합니다.
// 검증 결과가 유효하지 않으면 에러, 유효하면 fixed config를 반환합니다.
func ValidateAttributeConfig(typeName string, config datatypes.JSON) (datatypes.JSON, error) {
	engine, ok := GetEngine(typeName)
	if !ok {
		return nil, fmt.Errorf("no engine registered for type %s", typeName)
	}
	result := engine.ValidateConfig(config)
	if !result.IsValid {
		return nil, fmt.Errorf("invalid config for type '%s': %v", typeName, result.Errors)
	}
	return result.Fixed, nil
}

// ValidateAttributeValue는 주어진 속성 타입과 config, value에 대해 실제 값 검증 후, 정제된 결과 문자열을 반환합니다.
func ValidateAttributeValue(typeName string, config datatypes.JSON, value string) (string, error) {
	engine, ok := GetEngine(typeName)
	if !ok {
		return "", fmt.Errorf("no engine registered for type %s", typeName)
	}
	return engine.ValidateValue(value, config)
}

// DefaultConfigForType은 주어진 속성 타입에 대한 기본 config를 반환합니다.
func DefaultConfigForType(typeName string) (datatypes.JSON, error) {
	engine, ok := GetEngine(typeName)
	if !ok {
		return nil, fmt.Errorf("no engine registered for type %s", typeName)
	}
	return engine.DefaultConfig(), nil
}

// MergeConfigForType은 두 config를 병합합니다.
func MergeConfigForType(typeName string, current, new datatypes.JSON) (datatypes.JSON, error) {
	engine, ok := GetEngine(typeName)
	if !ok {
		return nil, fmt.Errorf("no engine registered for type %s", typeName)
	}
	return engine.MergeConfig(current, new), nil
}

// GetDisplayInfoForValue는 type, config, 저장값에 대한 사용자 표시 정보를 반환합니다.
func GetDisplayInfoForValue(typeName string, config datatypes.JSON, value string) (DisplayInfo, error) {
	engine, ok := GetEngine(typeName)
	if !ok {
		return DisplayInfo{}, fmt.Errorf("no engine registered for type %s", typeName)
	}
	return engine.GetDisplayInfo(value, config)
}

// lowercase는 주어진 문자열을 소문자로 변환합니다.
func lowercase(s string) string {
	return strings.ToLower(s)
}
