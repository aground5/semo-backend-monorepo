# 도메인 엔티티

이 디렉토리는 비즈니스 도메인 엔티티를 포함합니다. 도메인 엔티티는 비즈니스 핵심 개념을 표현하며 비즈니스 규칙과 로직을 캡슐화합니다.

## 역할과 책임

- 비즈니스 도메인 개념 표현
- 비즈니스 규칙 및 제약조건 적용
- 도메인 로직 캡슐화
- 외부 기술 의존성 없이 순수 비즈니스 로직만 포함

## 사용법

### 1. 엔티티 정의

비즈니스 개념에 집중하여 엔티티를 정의합니다:

```go
// user.go
package entity

import (
	"errors"
	"time"
)

type User struct {
	ID            string
	Username      string
	Email         string
	Password      string      // 비즈니스 로직에 필요한 경우에만
	AccountStatus string
	LastLoginTime *time.Time
}
```

### 2. 생성자 함수 구현

유효성 검사와 초기화 로직을 포함한 생성자 함수를 제공합니다:

```go
// NewUser 생성자 함수
func NewUser(username, email, password string) (*User, error) {
	if username == "" {
		return nil, errors.New("사용자 이름은 필수입니다")
	}
	
	if email == "" {
		return nil, errors.New("이메일은 필수입니다")
	}
	
	return &User{
		Username:      username,
		Email:         email,
		Password:      password,
		AccountStatus: "inactive",
	}, nil
}
```

### 3. 비즈니스 로직 메서드 추가

상태 변경이나 비즈니스 규칙을 처리하는 메서드를 구현합니다:

```go
// 계정 활성화
func (u *User) Activate() {
	u.AccountStatus = "active"
}

// 비밀번호 변경
func (u *User) ChangePassword(newPassword string) error {
	if newPassword == "" {
		return errors.New("새 비밀번호는 비어있을 수 없습니다")
	}
	
	if newPassword == u.Password {
		return errors.New("새 비밀번호는 이전 비밀번호와 달라야 합니다")
	}
	
	u.Password = newPassword
	return nil
}

// 로그인 시도 기록
func (u *User) RecordLogin() {
	now := time.Now()
	u.LastLoginTime = &now
}
```

### 4. 도메인 로직 검증

비즈니스 규칙을 검증하는 메서드를 추가합니다:

```go
// 계정이 활성 상태인지 확인
func (u *User) IsActive() bool {
	return u.AccountStatus == "active"
}

// 이메일 형식 검증
func (u *User) HasValidEmail() bool {
	// 이메일 유효성 검사 로직
	return strings.Contains(u.Email, "@")
}
```

## FAQ

### Q: 도메인 엔티티에 ID 필드를 포함해야 하나요?
A: 일반적으로는 포함합니다. ID는 엔티티의 식별자로서 비즈니스 로직에서도 필요할 수 있습니다. 그러나 ID 생성 로직은 인프라스트럭처 계층으로 분리하는 것이 좋습니다.

### Q: 엔티티 간의 관계는 어떻게 표현하나요?
A: 다른 엔티티에 대한 참조는 ID 필드로 표현하거나, 필요한 경우 객체 참조를 사용할 수 있습니다. 객체 참조를 사용할 때는 순환 참조에 주의해야 합니다.

```go
// ID 참조를 사용한 관계 표현
type Order struct {
	ID       string
	UserID   string  // User 엔티티 ID 참조
	Products []string // Product 엔티티 ID 목록
}
```

### Q: 데이터베이스 관련 필드(CreatedAt, UpdatedAt 등)는 포함하지 않나요?
A: 이러한 필드가 비즈니스 로직에 필요한 경우에만 포함합니다. 단순히 데이터 관리용이라면 데이터베이스 모델에만 포함하는 것이 좋습니다.

### Q: 유효성 검사는 어디서 해야 하나요?
A: 생성자 함수와 상태 변경 메서드에서 유효성 검사를 수행합니다. 이를 통해 항상 일관된 상태를 유지할 수 있습니다.

### Q: 도메인 이벤트는 어떻게 처리하나요?
A: 엔티티 내에서 이벤트를 발생시키고 수집하는 메커니즘을 구현할 수 있습니다:

```go
type User struct {
	// 기존 필드들...
	events []DomainEvent
}

func (u *User) AddEvent(event DomainEvent) {
	u.events = append(u.events, event)
}

func (u *User) GetAndClearEvents() []DomainEvent {
	events := u.events
	u.events = []DomainEvent{}
	return events
}

// 사용 예시
func (u *User) Activate() {
	u.AccountStatus = "active"
	u.AddEvent(UserActivatedEvent{UserID: u.ID, Time: time.Now()})
}
```