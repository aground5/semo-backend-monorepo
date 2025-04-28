package repository

import (
	"context"
	"time"
)

// CacheRepository는 캐시 저장소 접근을 위한 인터페이스입니다.
type CacheRepository interface {
	// Set 지정된 키에 값을 저장합니다.
	Set(ctx context.Context, key string, value string, expiration time.Duration) error

	// Get 지정된 키에 해당하는 값을 조회합니다.
	Get(ctx context.Context, key string) (string, error)

	// Delete 지정된 키를 삭제합니다.
	Delete(ctx context.Context, key string) error

	// SetMulti 여러 키-값 쌍을 한 번에 저장합니다.
	SetMulti(ctx context.Context, items map[string]string, expiration time.Duration) error

	// DeleteMulti 여러 키를 한 번에 삭제합니다.
	DeleteMulti(ctx context.Context, keys []string) error

	// IsNotFound는 주어진 에러가 키가 없음을 나타내는지 확인합니다.
	IsNotFound(err error) bool
}
