package errors

import (
	"go.uber.org/zap"
)

// LogError는 에러를 구조화된 로그로 기록합니다
func LogError(logger *zap.Logger, err error, msg string, fields ...zap.Field) {
	if err == nil {
		return
	}

	// 기본 필드
	allFields := make([]zap.Field, 0, len(fields)+2)
	allFields = append(allFields, zap.Error(err))

	// AppError에서 추가 정보 추출
	var appErr *AppError
	if As(err, &appErr) {
		allFields = append(allFields, zap.String("error_code", appErr.Code()))
	}

	// 추가 필드 병합
	allFields = append(allFields, fields...)

	// 로깅
	logger.Error(msg, allFields...)
}
