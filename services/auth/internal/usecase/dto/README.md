
# Domain Entity와 DTO의 차이점

## 개요
클린 아키텍처에서 Domain Entity와 DTO(Data Transfer Object)는 서로 다른 목적과 특성을 가집니다. 이 문서는 두 개념의 차이점과 각각의 용도를 설명합니다.

## Domain Entity

Domain Entity는 비즈니스 도메인의 핵심 개념과 규칙을 표현하는 객체입니다.

### 특징
- 비즈니스 도메인의 핵심 개념 표현
- 비즈니스 규칙과 유효성 검사 로직 포함
- 데이터베이스 구조와 독립적으로 설계
- 장기적으로 안정적인 구조 유지
- 비즈니스 메서드 포함

### 예시
```go
package entity

import "time"

type User struct {
    ID        string
    Email     string
    Password  string
    Role      string
    CreatedAt time.Time
    
    // 비즈니스 메서드
    ChangePassword(oldPassword, newPassword string) error
    HasPermission(action string) bool
}
```

## DTO (Data Transfer Object)

DTO는 계층 간 데이터 전송을 위한 단순 데이터 구조체입니다.

### 특징
- 데이터 전송 목적으로만 사용
- 비즈니스 로직 없이 데이터만 포함
- 특정 유스케이스나 API에 최적화된 구조
- 클라이언트 요구사항에 따라 변경 가능
- JSON/XML 직렬화를 위한 태그 포함

### 예시
```go
package dto

type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

type UserResponse struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
    // 비밀번호와 같은 민감 정보 제외
}
```

## 차이점

| 특성 | Domain Entity | DTO |
|------|--------------|-----|
| **목적** | 비즈니스 개념과 규칙 표현 | 계층 간 데이터 전송 |
| **포함 내용** | 모든 비즈니스 관련 데이터와 로직 | 특정 상황에 필요한 데이터만 |
| **메서드** | 비즈니스 로직 메서드 포함 | 일반적으로 getter/setter만 포함 |
| **사용 위치** | 도메인 레이어 내부 | 레이어 간 경계(API ↔ 유스케이스 ↔ 도메인) |
| **변경 이유** | 비즈니스 규칙 변경 시 | UI/API 요구사항 변경 시 |
| **의존성** | 외부 레이어에 의존하지 않음 | 사용 목적에 따라 의존성 가능 |
| **유효성 검사** | 도메인 규칙 기반 검증 | 입력 형식 검증 (API 요청 등) |

## 변환 패턴

일반적으로 유스케이스에서는 DTO와 Entity 간의 변환 로직을 포함합니다:

```go
// 유스케이스 내부
func (u *AuthUseCase) Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error) {
    // DTO → Entity 변환
    user, err := u.userRepo.FindByEmail(ctx, req.Email)
    if err != nil {
        return nil, err
    }
    
    // 비즈니스 로직 수행
    if !user.ValidatePassword(req.Password) {
        return nil, errors.New("인증 실패")
    }
    
    // Entity → DTO 변환
    return &dto.AuthResponse{
        User: &dto.UserResponse{
            ID:    user.ID,
            Email: user.Email,
            Name:  user.Name,
        },
        Token: tokenString,
    }, nil
}
```

## 결론

Entity와 DTO는 클린 아키텍처에서 각각 다른 역할을 수행합니다. 이들을 명확히 구분하여 사용함으로써 코드의 유지보수성을 높이고, 각 계층의 책임을 분리할 수 있습니다.
