package errors

import (
	"errors"
	"fmt"
)

// 표준 라이브러리 함수 재노출
var (
	New    = errors.New
	Unwrap = errors.Unwrap
	Is     = errors.Is
	As     = errors.As
)

// Error는 기본 에러 인터페이스를 확장합니다
type Error interface {
	error
	Code() string  // 에러 코드 반환
	Unwrap() error // 내부 에러 반환
}

// AppError는 기본 에러 구현체입니다
type AppError struct {
	code    string
	message string
	err     error
}

func (e *AppError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s", e.message, e.err.Error())
	}
	return e.message
}

func (e *AppError) Code() string {
	return e.code
}

func (e *AppError) Unwrap() error {
	return e.err
}

// NewAppError는 새 애플리케이션 에러를 생성합니다
func NewAppError(code string, message string, err error) *AppError {
	return &AppError{
		code:    code,
		message: message,
		err:     err,
	}
}

// Wrap은 기존 에러를 래핑합니다
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	// 기존 AppError인 경우 코드를 유지합니다
	var appErr *AppError
	if As(err, &appErr) {
		return NewAppError(appErr.Code(), message, err)
	}

	return NewAppError(ErrInternal, message, err)
}
