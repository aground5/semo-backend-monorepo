# 어댑터 계층

이 디렉토리는 API 서비스의 어댑터 계층을 포함하고 있습니다. 어댑터 계층은 외부 시스템과의 인터페이스를 담당합니다.

## 구조

- **handler**: HTTP 핸들러 및 API 엔드포인트
- **repository**: 데이터베이스 구현
- **middleware**: HTTP 미들웨어

## 사용 방법

### HTTP 핸들러 구현

HTTP 핸들러는 외부 요청을 처리하고 유스케이스와 연결합니다:

```go
package handler

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/usecase/item"
)

type ItemHandler struct {
	itemUseCase item.UseCase
}

func NewItemHandler(itemUseCase item.UseCase) *ItemHandler {
	return &ItemHandler{
		itemUseCase: itemUseCase,
	}
}

func (h *ItemHandler) Create(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 요청 처리
	item, err := h.itemUseCase.CreateItem(c.Request.Context(), req.Name, req.Contents, req.Type, req.CreatedBy, req.ParentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, item)
}

// 다른 핸들러 메서드 구현...
```

### 레포지토리 구현

레포지토리는 도메인 레포지토리 인터페이스의 구현체입니다:

```go
package repository

import (
	"context"
	
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/domain/repository"
	"gorm.io/gorm"
)

type itemRepository struct {
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) repository.ItemRepository {
	return &itemRepository{
		db: db,
	}
}

func (r *itemRepository) FindByID(ctx context.Context, id string) (*entity.Item, error) {
	var item ItemModel
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	
	return mapItemModelToEntity(&item), nil
}

// 다른 메서드 구현...
```

### 미들웨어 구현

미들웨어는 HTTP 요청 처리 과정에서 공통 기능을 제공합니다:

```go
package middleware

import (
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		
		// 토큰 검증 로직
		// ...
		
		c.Next()
	}
}
```

## 가이드라인

1. 어댑터 계층은 외부 시스템(데이터베이스, HTTP 등)과의 상호작용을 담당합니다.
2. 어댑터 계층은 도메인 계층이나 유스케이스 계층에 의존할 수 있지만, 그 반대는 아닙니다.
3. 어댑터는 도메인 모델과 외부 표현 간의 변환을 담당합니다.
4. 인프라스트럭처 종속성(데이터베이스, 메시징 등)은 어댑터 계층에서 구현합니다. 