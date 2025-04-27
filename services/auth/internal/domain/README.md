# 도메인 계층

이 디렉토리는 인증 서비스의 도메인 계층을 포함하고 있습니다. 도메인 계층은 비즈니스 엔티티와 비즈니스 규칙을 정의합니다.

## 구조

- **entity**: 핵심 비즈니스 객체 (사용자, 권한 등)
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

type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
    RoleGuest Role = "guest"
)

type User struct {
    ID           string    `json:"id"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`
    Role         Role      `json:"role"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

func NewUser(email, passwordHash string, role Role) (*User, error) {
    if email == "" {
        return nil, errors.New("이메일은 필수입니다")
    }
    
    if passwordHash == "" {
        return nil, errors.New("비밀번호 해시는 필수입니다")
    }
    
    if role == "" {
        role = RoleUser // 기본 역할 설정
    }
    
    now := time.Now()
    
    return &User{
        Email:        email,
        PasswordHash: passwordHash,
        Role:         role,
        CreatedAt:    now,
        UpdatedAt:    now,
    }, nil
}
```

### 레포지토리 인터페이스 정의

레포지토리 인터페이스는 도메인 엔티티에 대한 데이터 접근 계약을 정의합니다:

```go
package repository

import (
    "context"
    
    "github.com/your-org/semo-backend-monorepo/services/auth/internal/domain/entity"
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
4. 인증 및 권한 검사 로직은 도메인 서비스에 구현하세요. 