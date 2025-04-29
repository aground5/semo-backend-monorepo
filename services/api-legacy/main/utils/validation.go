package utils

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// ValidationError는 validation 에러를 표현하는 커스텀 에러 타입입니다.
type ValidationError struct {
	Msg  string
	Errs []string
}

func (e *ValidationError) Error() string {
	if len(e.Errs) == 0 {
		return e.Msg
	}
	return e.Msg + ": " + strings.Join(e.Errs, ", ")
}

func NewValidationError(msg string, errs []string) error {
	return &ValidationError{
		Msg:  msg,
		Errs: errs,
	}
}

// SafeJSONUnmarshal은 json.Unmarshal을 감싸 안전하게 처리하는 함수입니다.
func SafeJSONUnmarshal(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return NewValidationError("JSON unmarshal error", []string{err.Error()})
	}
	return nil
}

// SanitizeString은 입력 문자열의 앞뒤 공백을 제거하고 HTML 특수문자를 이스케이프 처리합니다.
func SanitizeString(input string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(strings.TrimSpace(input))
}

// ValidateColorString은 "#RRGGBB" 형식의 색상 문자열인지 검증합니다.
func ValidateColorString(s string) error {
	if len(s) != 7 || !strings.HasPrefix(s, "#") {
		return errors.New("invalid color string: must be # followed by 6 hex digits")
	}
	if _, err := hex.DecodeString(s[1:]); err != nil {
		return errors.New("invalid hex code in color string")
	}
	return nil
}

// ValidateHex은 ValidateColorString과 동일하게 동작합니다.
func ValidateHex(s string) error {
	return ValidateColorString(s)
}
