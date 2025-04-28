package errors

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ToHTTPStatus는 에러 코드를 HTTP 상태 코드로 변환합니다
func ToHTTPStatus(code string) int {
	httpStatus, _ := GetCodeMapping(code)
	return httpStatus
}

// ToHTTPError는 에러를 Echo HTTP 에러로 변환합니다
func ToHTTPError(err error) *echo.HTTPError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if As(err, &appErr) {
		httpStatus := ToHTTPStatus(appErr.Code())
		return echo.NewHTTPError(httpStatus, appErr.Error())
	}

	// Echo 에러인 경우 그대로 반환
	if echoErr, ok := err.(*echo.HTTPError); ok {
		return echoErr
	}

	// 기본 에러는 500으로 처리
	return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
}

// FromHTTPError는 Echo HTTP 에러를 내부 에러로 변환합니다
func FromHTTPError(err error) error {
	if err == nil {
		return nil
	}

	// 이미 AppError인 경우 그대로 반환
	var appErr *AppError
	if As(err, &appErr) {
		return err
	}

	// Echo 에러 처리
	if echoErr, ok := err.(*echo.HTTPError); ok {
		code := httpStatusToCode(echoErr.Code)
		var msg string
		if m, ok := echoErr.Message.(string); ok {
			msg = m
		} else {
			msg = "HTTP error"
		}
		return NewAppError(code, msg, nil)
	}

	// 기본 에러는 Internal로 처리
	return NewAppError(ErrInternal, err.Error(), err)
}

// httpStatusToCode는 HTTP 상태 코드를 내부 에러 코드로 변환합니다
func httpStatusToCode(status int) string {
	switch status {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusBadRequest:
		return ErrInvalidArgument
	case http.StatusUnauthorized:
		return ErrUnauthenticated
	case http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusConflict:
		return ErrConflict
	case http.StatusGatewayTimeout:
		return ErrTimeout
	case http.StatusNotImplemented:
		return ErrNotImplemented
	default:
		return ErrInternal
	}
}
