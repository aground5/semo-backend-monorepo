package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"semo-server/configs"
	"semo-server/internal/repositories"
)

// RedisCache handles caching of AI responses in Redis
type RedisCache struct {
	Logger *zap.Logger
}

// NewRedisCache creates a new RedisCache instance
func NewRedisCache(logger *zap.Logger) *RedisCache {
	return &RedisCache{
		Logger: logger,
	}
}

// Set stores a value in Redis with the given key prefix and data for hashing
func Set(ctx context.Context, keyPrefix string, data map[string]any, value interface{}) error {
	// Generate hash for the data
	cacheKey, err := generateCacheKey(keyPrefix, data)
	if err != nil {
		return fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Serialize value to JSON
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Store in Redis with no expiration
	if err := repositories.DBS.Redis.Set(ctx, cacheKey, jsonData, 0).Err(); err != nil {
		return fmt.Errorf("failed to store value in Redis: %w", err)
	}

	return nil
}

// Get retrieves a value from Redis with the given key prefix and data for hashing
func Get(ctx context.Context, keyPrefix string, data map[string]any, value interface{}) (bool, error) {
	// Generate hash for the data
	cacheKey, err := generateCacheKey(keyPrefix, data)
	if err != nil {
		return false, nil // Skip cache on error
	}

	// Get cached data from Redis
	cachedData, err := repositories.DBS.Redis.Get(ctx, cacheKey).Result()
	if err != nil {
		// If error is not a nil reply (key not found), log it
		if err.Error() != "redis: nil" {
			configs.Logger.Warn("Failed to get cached value from Redis", zap.Error(err))
		}
		return false, nil
	}

	// Deserialize JSON to value
	if err := json.Unmarshal([]byte(cachedData), value); err != nil {
		configs.Logger.Warn("Failed to deserialize cached value", zap.Error(err))
		return false, nil
	}

	configs.Logger.Info("Cache hit", zap.String("cacheKey", cacheKey))
	return true, nil
}

// SetWithExpiration stores a value in Redis with the given key prefix, data for hashing, and expiration
func SetWithExpiration(ctx context.Context, keyPrefix string, data map[string]any, value interface{}, expiration time.Duration) error {
	// Generate hash for the data
	cacheKey, err := generateCacheKey(keyPrefix, data)
	if err != nil {
		return fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Serialize value to JSON
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Store in Redis with the specified expiration
	if err := repositories.DBS.Redis.Set(ctx, cacheKey, jsonData, expiration).Err(); err != nil {
		return fmt.Errorf("failed to store value in Redis: %w", err)
	}

	return nil
}

// ClearByPrefix clears all cached responses from Redis with the given prefix
func ClearByPrefix(ctx context.Context, keyPrefix string) error {
	// Get all keys with the given prefix
	keys, err := repositories.DBS.Redis.Keys(ctx, keyPrefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys with prefix %s: %w", keyPrefix, err)
	}

	// If there are keys to delete
	if len(keys) > 0 {
		// Delete all matching keys
		if err := repositories.DBS.Redis.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to clear cache with prefix %s: %w", keyPrefix, err)
		}
	}

	configs.Logger.Info("Cache cleared", zap.String("prefix", keyPrefix), zap.Int("keysDeleted", len(keys)))
	return nil
}

// generateCacheKey creates a unique key from the data map and prefix
func generateCacheKey(prefix string, data map[string]any) (string, error) {
	// Convert data to JSON for consistent serialization
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data for cache key: %w", err)
	}

	// Create a hash of the JSON data
	hash := sha256.Sum256(jsonData)
	hashStr := hex.EncodeToString(hash[:])[:16]
	
	return fmt.Sprintf("%s:%s", prefix, hashStr), nil
}