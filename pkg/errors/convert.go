package errors

// CodePair는 프레임워크 간 코드 매핑을 위한 구조체입니다
type CodePair struct {
	HTTPStatus int
	GRPCCode   int
}

// 코드 매핑 테이블
var codeMapping = map[string]CodePair{
	ErrInternal:        {500, 13}, // Internal Server Error, INTERNAL
	ErrNotFound:        {404, 5},  // Not Found, NOT_FOUND
	ErrInvalidArgument: {400, 3},  // Bad Request, INVALID_ARGUMENT
	ErrUnauthenticated: {401, 16}, // Unauthorized, UNAUTHENTICATED
	ErrUnauthorized:    {403, 7},  // Forbidden, PERMISSION_DENIED
	ErrConflict:        {409, 6},  // Conflict, ALREADY_EXISTS
	ErrTimeout:         {504, 4},  // Gateway Timeout, DEADLINE_EXCEEDED
	ErrNotImplemented:  {501, 12}, // Not Implemented, UNIMPLEMENTED
}

// GetCodeMapping은 특정 에러 코드에 대한 HTTP 및 gRPC 코드 매핑을 반환합니다
func GetCodeMapping(code string) (int, int) {
	if pair, ok := codeMapping[code]; ok {
		return pair.HTTPStatus, pair.GRPCCode
	}
	return 500, 13 // 기본값으로 Internal Server Error
}
