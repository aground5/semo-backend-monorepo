# 유스케이스 계층

이 디렉토리는 API 서비스의 유스케이스 계층을 포함하고 있습니다. 유스케이스 계층은 애플리케이션 특정 비즈니스 규칙을 정의합니다.

## 구조

- **item**: 아이템(태스크, 프로젝트) 관련 유스케이스
- **file**: 파일 관련 유스케이스
- **team**: 팀 관련 유스케이스
- **profile**: 프로필 관련 유스케이스

## 사용 방법

### 유스케이스 인터페이스 정의

유스케이스 인터페이스는 애플리케이션의 특정 기능을 정의합니다:

```go
package usecase

import (
    "context"
    
    "github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/domain/entity"
)

type ItemUseCase interface {
    CreateItem(ctx context.Context, name, contents string, itemType entity.ItemType, createdBy string, parentID *string) (*entity.Item, error)
    GetItemByID(ctx context.Context, id string) (*entity.Item, error)
    GetChildItems(ctx context.Context, parentID string, limit, offset int) ([]*entity.Item, error)
    UpdateItem(ctx context.Context, id, name, contents string) (*entity.Item, error)
    DeleteItem(ctx context.Context, id string) error
}
```

### 유스케이스 구현

유스케이스 구현은 비즈니스 로직을 캡슐화합니다:

```go
package usecase

import (
    "context"
    
    "github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/domain/entity"
    "github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/domain/repository"
)

type itemUseCase struct {
    itemRepo repository.ItemRepository
    activityRepo repository.ActivityRepository
}

func NewItemUseCase(itemRepo repository.ItemRepository, activityRepo repository.ActivityRepository) ItemUseCase {
    return &itemUseCase{
        itemRepo: itemRepo,
        activityRepo: activityRepo,
    }
}

func (uc *itemUseCase) CreateItem(ctx context.Context, name, contents string, itemType entity.ItemType, createdBy string, parentID *string) (*entity.Item, error) {
    item, err := entity.NewItem(name, contents, itemType, createdBy, parentID)
    if err != nil {
        return nil, err
    }
    
    if err := uc.itemRepo.Create(ctx, item); err != nil {
        return nil, err
    }
    
    // 활동 기록
    activity := &repository.ActivityInfo{
        Type:        repository.ActivityTypeCreate,
        ItemID:      item.ID,
        ProfileID:   createdBy,
        Description: "아이템 생성",
    }
    
    uc.activityRepo.Create(ctx, activity)
    
    return item, nil
}

// 다른 메서드 구현...
```

## 가이드라인

1. 유스케이스 계층은 도메인 엔티티와 레포지토리 인터페이스를 사용합니다.
2. 유스케이스는 하나 이상의 레포지토리를 조합하여 복잡한 비즈니스 로직을 구현합니다.
3. 외부 시스템과의 상호작용(이메일 발송, 외부 API 호출 등)은 어댑터 계층으로 위임해야 합니다.
4. 트랜잭션 관리와 같은 기술적 세부사항은 인프라스트럭처 계층에서 처리합니다. 