package errors

// 공통 에러 코드 정의
const (
	// 일반적인 에러 코드
	ErrInternal        = "INTERNAL"
	ErrNotFound        = "NOT_FOUND"
	ErrInvalidArgument = "INVALID_ARGUMENT"
	ErrUnauthenticated = "UNAUTHENTICATED"
	ErrUnauthorized    = "UNAUTHORIZED"
	ErrConflict        = "CONFLICT"
	ErrTimeout         = "TIMEOUT"
	ErrNotImplemented  = "NOT_IMPLEMENTED"
)
