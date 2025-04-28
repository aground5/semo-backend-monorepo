package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"go.uber.org/zap"
)

// RedisConfig Redis 설정
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// RedisRepository Redis 캐시 저장소 구현체
type RedisRepository struct {
	client *redis.Client
}

// NewRedisClient Redis 클라이언트 생성
func NewRedisClient(cfg RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		config.AppConfig.Logger.Error("Redis 연결 실패",
			zap.Error(err),
		)
		return nil, fmt.Errorf("Redis 연결 실패: %w", err)
	}

	config.AppConfig.Logger.Info("Redis 연결 성공",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
	)

	return client, nil
}

// NewRedisRepository Redis 저장소 생성
func NewRedisRepository(client *redis.Client) repository.CacheRepository {
	return &RedisRepository{
		client: client,
	}
}

// Set 키-값 저장
func (r *RedisRepository) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	if err := r.client.Set(ctx, key, value, expiration).Err(); err != nil {
		config.AppConfig.Logger.Error("Redis Set 실패",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// Get 키로 값 조회
func (r *RedisRepository) Get(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", err // 키가 없는 경우
		}
		config.AppConfig.Logger.Error("Redis Get 실패",
			zap.String("key", key),
			zap.Error(err),
		)
		return "", err
	}
	return value, nil
}

// Delete 키 삭제
func (r *RedisRepository) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		config.AppConfig.Logger.Error("Redis Delete 실패",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// SetMulti 여러 키-값 쌍 설정
func (r *RedisRepository) SetMulti(ctx context.Context, items map[string]string, expiration time.Duration) error {
	// 파이프라인 사용하여 한 번에 처리
	pipe := r.client.Pipeline()
	for key, value := range items {
		pipe.Set(ctx, key, value, expiration)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		config.AppConfig.Logger.Error("Redis SetMulti 실패",
			zap.Any("keys", items),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// DeleteMulti 여러 키 한 번에 삭제
func (r *RedisRepository) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		config.AppConfig.Logger.Error("Redis DeleteMulti 실패",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// IsNotFound 키가 존재하지 않는 에러인지 확인
func (r *RedisRepository) IsNotFound(err error) bool {
	return err == redis.Nil
}
