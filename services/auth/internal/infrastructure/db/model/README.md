# 데이터베이스 모델

이 디렉토리는 데이터베이스와 매핑되는 ORM 모델을 포함합니다. 이 모델들은 인프라스트럭처 계층에 속하며, 데이터 지속성을 처리합니다.

## 역할과 책임

- 데이터베이스 테이블 구조를 정의
- ORM 관련 메타데이터와 어노테이션 관리 (태그, 관계 등)
- 소프트 삭제, 타임스탬프 등 데이터베이스 관련 기능 처리
- 데이터 매핑 관련 로직 포함

## 사용법

### 1. 모델 정의

```go
// user_model.go
package model

import (
	"gorm.io/gorm"
	"time"
)

type UserModel struct {
	ID                string         `gorm:"type:char(12);primaryKey"`
	Username          string         `gorm:"size:100;not null"`
	Email             string         `gorm:"size:250;not null;uniqueIndex"`
	Password          string         `gorm:"size:250;not null"`
	// 기타 필드...
	CreatedAt         time.Time      `gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// TableName 테이블 이름 지정
func (UserModel) TableName() string {
	return "users"
}
```

### 2. 도메인 변환 함수 구현

레포지토리 구현체에서 사용할 변환 함수를 정의합니다:

```go
// 변환 함수 (repository/impl 패키지에 구현)
func toUserModel(entity *entity.User) *model.UserModel {
	return &model.UserModel{
		ID:            entity.ID,
		Username:      entity.Username,
		Email:         entity.Email,
		AccountStatus: entity.AccountStatus,
		// 다른 필드 매핑...
	}
}

func toUserEntity(model *model.UserModel) *entity.User {
	return &entity.User{
		ID:            model.ID,
		Username:      model.Username,
		Email:         model.Email,
		AccountStatus: model.AccountStatus,
		// 필요한 필드만 매핑...
	}
}
```

### 3. 마이그레이션 설정

애플리케이션 시작 시 DB 마이그레이션에 모델을 등록합니다:

```go
// infrastructure/db/migration.go
func SetupDatabase(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.UserModel{},
		&model.TokenModel{},
		// 다른 모델들...
	)
}
```

## FAQ

### Q: 도메인 엔티티와 DB 모델을 분리하는 이유는 무엇인가요?
A: 도메인 엔티티는 비즈니스 로직에 집중하고, DB 모델은 데이터 저장에 집중하여 관심사를 분리합니다. 또한 도메인 엔티티가 특정 ORM이나 데이터베이스 기술에 의존하지 않게 하여 유연성을 높입니다.

### Q: 모든 도메인 엔티티에 대응하는 DB 모델이 필요한가요?
A: 반드시 그렇지는 않습니다. 일부 도메인 엔티티는 영속화가 필요 없거나, 여러 테이블의 조합일 수 있습니다. 필요에 따라 모델을 설계하세요.

### Q: 관계 매핑은 어떻게 처리하나요?
A: DB 모델에서는 GORM의 관계 태그를 사용하여 정의하고, 도메인 엔티티에서는 ID 참조나 별도의 로딩 메커니즘을 통해 처리합니다.

```go
// DB 모델에서의 관계 정의
type TokenGroupModel struct {
	ID     uint       `gorm:"primaryKey"`
	UserID string     `gorm:"type:char(12);index"`
	Tokens []TokenModel `gorm:"foreignKey:GroupID"`
}
```

### Q: 트랜잭션은 어떻게 처리하나요?
A: 레포지토리 구현체에서 DB 모델을 사용할 때 트랜잭션을 처리합니다. 레포지토리 인터페이스에 트랜잭션 관련 메서드를 추가하거나, 서비스 레이어에서 트랜잭션을 관리할 수 있습니다.