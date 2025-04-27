package messaging

import (
	"context"
	"fmt"

	"github.com/your-org/semo-backend-monorepo/pkg/messaging"
	"github.com/your-org/semo-backend-monorepo/services/notification/internal/domain/entity"
)

// NotificationPublisher 알림 발행 인터페이스
type NotificationPublisher interface {
	PublishNotification(ctx context.Context, notification *entity.Notification) error
	Close() error
}

// redisNotificationPublisher Redis 기반 알림 발행자
type redisNotificationPublisher struct {
	redisClient messaging.RedisClient
	channel     string
}

// NewRedisNotificationPublisher Redis 알림 발행자 생성
func NewRedisNotificationPublisher(client messaging.RedisClient, channel string) NotificationPublisher {
	return &redisNotificationPublisher{
		redisClient: client,
		channel:     channel,
	}
}

// PublishNotification 알림 발행
func (p *redisNotificationPublisher) PublishNotification(ctx context.Context, notification *entity.Notification) error {
	if notification == nil {
		return fmt.Errorf("알림 데이터가 없습니다")
	}

	// 사용자별 채널
	userChannel := fmt.Sprintf("%s:%s", p.channel, notification.UserID)

	// 알림 발행
	if err := p.redisClient.Publish(ctx, userChannel, notification); err != nil {
		return fmt.Errorf("알림 발행 실패: %w", err)
	}

	// 모든 알림에 대한 채널에도 발행
	if err := p.redisClient.Publish(ctx, p.channel, notification); err != nil {
		return fmt.Errorf("전체 채널 알림 발행 실패: %w", err)
	}

	return nil
}

// Close 발행자 종료
func (p *redisNotificationPublisher) Close() error {
	return p.redisClient.Close()
}
