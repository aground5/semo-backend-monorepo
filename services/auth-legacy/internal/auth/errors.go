package auth

import (
	"errors"
	"fmt"
)

// AuthError는 인증 관련 오류를 표현하는 사용자 정의 오류 타입
type AuthError struct {
	Code    string
	Message string
	Err     error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// 오류 생성 헬퍼 함수
func NewAuthError(code, message string) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
	}
}

func NewAuthErrorWithCause(code, message string, cause error) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Err:     cause,
	}
}

// 오류 검사 헬퍼 함수
func IsAuthError(err error, code string) bool {
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.Code == code
	}
	return false
}