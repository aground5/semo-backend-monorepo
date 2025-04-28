# 모노레포 에러 시스템 관리 지침

## 1. 기본 원칙

### 1.1. 일관성
- 모든 서비스에서 동일한 에러 처리 패턴을 사용합니다.
- 에러 코드, 포맷, 로깅 방식이 일관되어야 합니다.

### 1.2. 명확성
- 에러 메시지는 구체적이고 행동 가능해야 합니다.
- 에러 코드는 의미를 명확히 전달해야 합니다.

### 1.3. 계층화
- 공통 에러와 서비스별 에러를 명확히 구분합니다.
- 각 계층에서 적절한 추상화를 유지합니다.

### 1.4. 확장성
- 새로운 에러 타입과 코드를 쉽게 추가할 수 있어야 합니다.
- 프레임워크 의존성과 비즈니스 로직을 분리합니다.

## 2. 디렉토리 구조

```
monorepo/
├── pkg/
│   └── errors/            # 공통 에러 패키지
│       ├── error.go       # 기본 에러 인터페이스 및 타입
│       ├── codes.go       # 공통 에러 코드 정의
│       ├── convert.go     # 프레임워크 간 에러 변환 유틸리티
│       ├── http.go        # HTTP 에러 변환 유틸리티
│       ├── grpc.go        # gRPC 에러 변환 유틸리티
│       └── logging.go     # Zap 로거 통합 유틸리티
│
└── services/
    └── userservice/
        └── internal/
            └── errors/    # 서비스별 에러 패키지
                ├── codes.go  # 서비스별 에러 코드
                ├── errors.go # 서비스별 에러 생성 함수
                └── handlers.go # 서비스별 에러 핸들러
```

## 3. 공통 에러 패키지 (pkg/errors)

### 3.1. 기본 에러 인터페이스 및 타입 (error.go)

```go
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
    Code() string     // 에러 코드 반환
    Unwrap() error    // 내부 에러 반환
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
```

### 3.2. 공통 에러 코드 (codes.go)

```go
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
```

### 3.3. 프레임워크 변환 유틸리티 (convert.go)

```go
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
```

### 3.4. HTTP 에러 변환 (http.go)

```go
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
```

### 3.5. gRPC 에러 변환 (grpc.go)

```go
package errors

import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// ToGRPCStatus는 에러 코드를 gRPC 상태 코드로 변환합니다
func ToGRPCStatus(code string) codes.Code {
    _, grpcCode := GetCodeMapping(code)
    return codes.Code(grpcCode)
}

// ToGRPCError는 에러를 gRPC 에러로 변환합니다
func ToGRPCError(err error) error {
    if err == nil {
        return nil
    }
    
    var appErr *AppError
    if As(err, &appErr) {
        grpcCode := ToGRPCStatus(appErr.Code())
        return status.Error(grpcCode, appErr.Error())
    }
    
    // 이미 gRPC 에러인 경우 그대로 반환
    if _, ok := status.FromError(err); ok {
        return err
    }
    
    // 기본 에러는 Internal로 처리
    return status.Error(codes.Internal, err.Error())
}

// FromGRPCError는 gRPC 에러를 내부 에러로 변환합니다
func FromGRPCError(err error) error {
    if err == nil {
        return nil
    }
    
    // 이미 AppError인 경우 그대로 반환
    var appErr *AppError
    if As(err, &appErr) {
        return err
    }
    
    // gRPC 에러 처리
    if st, ok := status.FromError(err); ok {
        code := grpcStatusToCode(st.Code())
        return NewAppError(code, st.Message(), nil)
    }
    
    // 기본 에러는 Internal로 처리
    return NewAppError(ErrInternal, err.Error(), err)
}

// grpcStatusToCode는 gRPC 상태 코드를 내부 에러 코드로 변환합니다
func grpcStatusToCode(code codes.Code) string {
    switch code {
    case codes.NotFound:
        return ErrNotFound
    case codes.InvalidArgument:
        return ErrInvalidArgument
    case codes.Unauthenticated:
        return ErrUnauthenticated
    case codes.PermissionDenied:
        return ErrUnauthorized
    case codes.AlreadyExists:
        return ErrConflict
    case codes.DeadlineExceeded:
        return ErrTimeout
    case codes.Unimplemented:
        return ErrNotImplemented
    default:
        return ErrInternal
    }
}
```

### 3.6. 로깅 통합 (logging.go)

```go
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
```

## 4. 서비스별 에러 패키지 (services/*/internal/errors)

### 4.1. 서비스별 에러 코드 (codes.go)

```go
package errors

import (
    "github.com/your-org/monorepo/pkg/errors"
)

// UserService 에러 코드
const (
    // 서비스 접두사로 에러 코드 네임스페이스 분리
    ErrUserNotFound      = "USER_NOT_FOUND"
    ErrUserAlreadyExists = "USER_ALREADY_EXISTS"
    ErrInvalidEmail      = "USER_INVALID_EMAIL"
    ErrInvalidPassword   = "USER_INVALID_PASSWORD"
    ErrEmailTaken        = "USER_EMAIL_TAKEN"
)

// 공통 에러 코드 재노출
var (
    ErrInternal        = errors.ErrInternal
    ErrInvalidArgument = errors.ErrInvalidArgument
    // 필요한 공통 에러 코드만 선택적으로 재노출
)
```

### 4.2. 서비스별 에러 생성 함수 (errors.go)

```go
package errors

import (
    "fmt"
    
    "github.com/your-org/monorepo/pkg/errors"
)

// 공통 에러 함수 재노출
var (
    NewAppError = errors.NewAppError
    Wrap        = errors.Wrap
    Is          = errors.Is
    As          = errors.As
)

// UserNotFoundError는 사용자를 찾을 수 없을 때 사용됩니다
func UserNotFoundError(userID string) error {
    return NewAppError(
        ErrUserNotFound,
        fmt.Sprintf("User not found with ID: %s", userID),
        nil,
    )
}

// InvalidEmailError는 이메일 형식이 잘못된 경우 사용됩니다
func InvalidEmailError(email string) error {
    return NewAppError(
        ErrInvalidEmail,
        fmt.Sprintf("Invalid email format: %s", email),
        nil,
    )
}

// UserAlreadyExistsError는 이미 존재하는 사용자를 생성하려 할 때 사용됩니다
func UserAlreadyExistsError(email string) error {
    return NewAppError(
        ErrUserAlreadyExists,
        fmt.Sprintf("User with email %s already exists", email),
        nil,
    )
}
```

### 4.3. 서비스별 에러 핸들러 (handlers.go)

```go
package errors

import (
    "net/http"
    
    "github.com/labstack/echo/v4"
    "github.com/your-org/monorepo/pkg/errors"
    "go.uber.org/zap"
)

// ErrorResponse는 클라이언트에게 반환되는 에러 응답 구조체입니다
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// HTTPErrorHandler는 Echo 프레임워크용 에러 핸들러를 생성합니다
func HTTPErrorHandler(logger *zap.Logger) echo.HTTPErrorHandler {
    return func(err error, c echo.Context) {
        // 내부 에러 타입으로 변환
        var appErr *errors.AppError
        var code string
        var message string
        
        if errors.As(err, &appErr) {
            code = appErr.Code()
            message = appErr.Error()
        } else if echoErr, ok := err.(*echo.HTTPError); ok {
            // Echo 에러 처리
            code = httpStatusToCode(echoErr.Code)
            if msg, ok := echoErr.Message.(string); ok {
                message = msg
            } else {
                message = "HTTP error"
            }
        } else {
            // 기타 에러는 내부 에러로 처리
            code = errors.ErrInternal
            message = err.Error()
        }
        
        // 상태 코드 결정
        status := getHTTPStatusForCode(code)
        
        // 로깅
        errors.LogError(logger, err, "HTTP request error",
            zap.String("code", code),
            zap.Int("status", status),
            zap.String("method", c.Request().Method),
            zap.String("path", c.Request().URL.Path),
        )
        
        // 응답 반환
        c.JSON(status, ErrorResponse{
            Code:    code,
            Message: getClientMessage(message, status),
        })
    }
}

// getHTTPStatusForCode는 에러 코드에 대한 HTTP 상태 코드를 반환합니다
func getHTTPStatusForCode(code string) int {
    // 서비스별 에러 코드에 대한 상태 코드 매핑
    switch code {
    case ErrUserNotFound:
        return http.StatusNotFound
    case ErrUserAlreadyExists, ErrEmailTaken:
        return http.StatusConflict
    case ErrInvalidEmail, ErrInvalidPassword:
        return http.StatusBadRequest
    default:
        // 기본 매핑은 공통 패키지 사용
        return errors.ToHTTPStatus(code)
    }
}

// getClientMessage는 에러 메시지를 클라이언트에 적합한 형태로 변환합니다
func getClientMessage(message string, status int) string {
    // 프로덕션 환경에서는 500 에러의 상세 내용을 숨길 수 있음
    if status == http.StatusInternalServerError && isProd() {
        return "Internal server error occurred. Please try again later."
    }
    return message
}

// isProd는 현재 환경이 프로덕션인지 확인합니다
func isProd() bool {
    // 환경 설정에 따라 구현
    return false
}
```

## 5. 적용 가이드라인

### 5.1. 에러 생성

- 가능한 한 구체적인 에러 타입을 사용합니다.
- 에러 메시지는 문제와 가능한 해결책을 명확히 설명합니다.
- 내부 에러를 래핑하여 컨텍스트를 추가합니다.

```go
// 좋은 예
if user == nil {
    return errors.UserNotFoundError(userID)
}

// 좋은 예 (에러 래핑)
user, err := repo.FindByID(ctx, userID)
if err != nil {
    return errors.Wrap(err, "failed to find user")
}
```

### 5.2. 에러 처리

- 에러 타입이나 코드를 확인하여 적절히 처리합니다.
- 로깅은 적절한 컨텍스트와 함께 수행합니다.
- 클라이언트에게는 필요한 정보만 노출합니다.

```go
func handleGetUser(c echo.Context) error {
    userID := c.Param("id")
    user, err := service.GetUser(c.Request().Context(), userID)
    
    if err != nil {
        // 에러 타입 검사
        var appErr *errors.AppError
        if errors.As(err, &appErr) && appErr.Code() == errors.ErrUserNotFound {
            // 404 처리
            return echo.NewHTTPError(http.StatusNotFound, "User not found")
        }
        // 기타 에러는 서버 에러로 처리
        logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user")
    }
    
    return c.JSON(http.StatusOK, user)
}
```

### 5.3. gRPC 서비스 적용

- gRPC 서비스에서는 상태 코드를 활용합니다.
- 클라이언트에 반환할 상세 정보를 제어합니다.

```go
func (s *userService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    user, err := s.repo.FindByID(ctx, req.UserId)
    if err != nil {
        // 내부 에러를 gRPC 에러로 변환
        return nil, errors.ToGRPCError(err)
    }
    
    return convertUserToProto(user), nil
}
```

### 5.4. 에러 코드 추가

- 새 에러 코드는 명확한 네이밍과 문서화와 함께 추가합니다.
- 서비스별 에러 코드는 서비스명 접두어를 사용합니다.

```go
// 새 에러 코드 추가 (서비스별 errors/codes.go)
const (
    // 기존 코드...
    
    // 결제 관련 에러
    ErrPaymentFailed        = "USER_PAYMENT_FAILED"
    ErrInsufficientBalance  = "USER_INSUFFICIENT_BALANCE"
)

// 새 에러 생성 함수 (서비스별 errors/errors.go)
func PaymentFailedError(reason string) error {
    return NewAppError(
        ErrPaymentFailed,
        fmt.Sprintf("Payment failed: %s", reason),
        nil,
    )
}
```

## 6. 모범 사례

### 6.1. 구체적인 에러 반환

- 가능한 한 구체적인 에러 타입과 메시지를 사용합니다.
- 에러 메시지는 문제와 가능한 해결책을 설명해야 합니다.

```go
// 나쁜 예
return errors.NewAppError(errors.ErrInvalidArgument, "invalid input", nil)

// 좋은 예
return errors.InvalidEmailError(email)
```

### 6.2. 적절한 로깅

- 에러 로깅은 개발자가 문제를 진단하기에 충분한 정보를 포함해야 합니다.
- 민감한 정보는 로그에서 제외해야 합니다.

```go
// 좋은 예
errors.LogError(logger, err, "User registration failed",
    zap.String("email", sanitizeEmail(email)),
    zap.String("registration_source", source),
)
```

### 6.3. 에러 처리 테스트

- 에러 처리 로직에 대한 단위 테스트를 작성합니다.
- 특히 에러 변환 및 핸들링 로직을 테스트합니다.

```go
func TestToHTTPStatus(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected int
    }{
        {"not_found", errors.ErrNotFound, http.StatusNotFound},
        {"invalid_argument", errors.ErrInvalidArgument, http.StatusBadRequest},
        // 기타 테스트 케이스
    }
    
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            status := errors.ToHTTPStatus(tc.code)
            if status != tc.expected {
                t.Errorf("Expected status %d, got %d", tc.expected, status)
            }
        })
    }
}
```

## 7. 주의사항

### 7.1. 과도한 추상화 피하기

- 필요 이상으로 복잡한 에러 시스템은 유지보수를 어렵게 만듭니다.
- 팀 규모와 프로젝트 복잡도에 맞게 적절한 수준의 추상화를 선택합니다.

### 7.2. 에러 메시지 국제화

- 사용자에게 표시되는 에러 메시지는 국제화가 필요할 수 있습니다.
- 메시지 키를 사용하여 국제화를 지원하는 방식을 고려합니다.

### 7.3. 보안 고려사항

- 내부 에러 세부 정보가 외부에 노출되지 않도록 합니다.
- 프로덕션 환경에서는 민감한 정보를 포함한 에러 메시지를 숨깁니다.

## 8. 마이그레이션 전략

기존 코드에서 새 에러 시스템으로 마이그레이션할 때는 점진적 접근이 권장됩니다:

1. 공통 에러 패키지 구현
2. 신규 서비스에 새 에러 시스템 적용
3. 기존 서비스의 새로운 기능에 적용
4. 핵심 기능부터 점진적으로 마이그레이션

## 9. 정기적인 검토

에러 시스템은 정기적으로 검토하고 개선해야 합니다:

- 새로운 에러 패턴 확인
- 중복되거나 불필요한 에러 코드 제거
- 에러 처리 모범 사례 공유
- 로깅 및 모니터링 개선

---

이 지침은 10명 규모의 팀에서 모노레포 환경에서 다양한 서비스와 프레임워크(Echo, gRPC)를 사용할 때 일관된 에러 처리 시스템을 구축하기 위한 기초를 제공합니다. 프로젝트의 요구사항에 맞게 조정하여 사용하세요.