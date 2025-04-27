# 도메인 계층

이 디렉토리는 API 서비스의 도메인 계층을 포함하고 있습니다. 도메인 계층은 비즈니스 엔티티와 비즈니스 규칙을 정의합니다.

## 구조

- **entity**: 핵심 비즈니스 객체 (사용자, 상품 등)
- **repository**: 데이터 저장소 인터페이스

## 사용 방법

### 엔티티 정의

엔티티는 비즈니스 객체를 표현하며, 유효성 검사 메서드를 포함합니다:

```go
package entity

import (
    "time"
    "errors"
)

type User struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func NewUser(username, email string) (*User, error) {
    if username == "" {
        return nil, errors.New("사용자 이름은 필수입니다")
    }
    
    if email == "" {
        return nil, errors.New("이메일은 필수입니다")
    }
    
    now := time.Now()
    
    return &User{
        Username:  username,
        Email:     email,
        CreatedAt: now,
        UpdatedAt: now,
    }, nil
}
```

### 레포지토리 인터페이스 정의

레포지토리 인터페이스는 도메인 엔티티에 대한 데이터 접근 계약을 정의합니다:

```go
package repository

import (
    "context"
    
    "github.com/your-org/semo-backend-monorepo/services/api/internal/domain/entity"
)

type UserRepository interface {
    Create(ctx context.Context, user *entity.User) error
    GetByID(ctx context.Context, id string) (*entity.User, error)
    GetByEmail(ctx context.Context, email string) (*entity.User, error)
    Update(ctx context.Context, user *entity.User) error
    Delete(ctx context.Context, id string) error
}
```

## 가이드라인

1. 도메인 계층은 외부 종속성(데이터베이스, 외부 API 등)을 가져오지 않아야 합니다.
2. 도메인 계층은 비즈니스 규칙만 포함해야 합니다.
3. 레포지토리 인터페이스는 도메인 엔티티만 사용해야 합니다.
4. 복잡한 비즈니스 규칙은 도메인 서비스로 분리하세요. 