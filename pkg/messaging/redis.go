package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient Redis 클라이언트 인터페이스
type RedisClient interface {
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channel string) (<-chan Message, error)
	Close() error
}

// Message 메시지 구조체
type Message struct {
	Channel string
	Payload []byte
	Time    time.Time
}

// redisClient Redis 클라이언트 구현체
type redisClient struct {
	client *redis.Client
}

// NewRedisClient Redis 클라이언트 생성
func NewRedisClient(addr, password string, db int) (RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Redis 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 연결 실패: %w", err)
	}

	return &redisClient{
		client: client,
	}, nil
}

// Publish 메시지 발행
func (r *redisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("메시지 직렬화 실패: %w", err)
	}

	return r.client.Publish(ctx, channel, payload).Err()
}

// Subscribe 채널 구독
func (r *redisClient) Subscribe(ctx context.Context, channel string) (<-chan Message, error) {
	pubsub := r.client.Subscribe(ctx, channel)

	// 구독 확인
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("채널 구독 실패: %w", err)
	}

	messageCh := make(chan Message)
	go func() {
		defer close(messageCh)
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				messageCh <- Message{
					Channel: msg.Channel,
					Payload: []byte(msg.Payload),
					Time:    time.Now(),
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return messageCh, nil
}

// Close Redis 클라이언트 종료
func (r *redisClient) Close() error {
	return r.client.Close()
}
