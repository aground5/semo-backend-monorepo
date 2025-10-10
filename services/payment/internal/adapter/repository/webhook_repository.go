package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WebhookRepository handles webhook event storage and processing
type WebhookRepository interface {
	SaveEvent(ctx context.Context, eventID, eventType string, data json.RawMessage) error
	GetEvent(ctx context.Context, eventID string) (*model.StripeWebhookEvent, error)
	MarkProcessed(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string, err error) error
	GetPendingEvents(ctx context.Context, limit int) ([]*model.StripeWebhookEvent, error)
}

type webhookRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewWebhookRepository creates a new webhook repository
func NewWebhookRepository(db *gorm.DB, logger *zap.Logger) WebhookRepository {
	return &webhookRepository{
		db:     db,
		logger: logger,
	}
}

// SaveEvent saves a new webhook event
func (r *webhookRepository) SaveEvent(ctx context.Context, eventID, eventType string, data json.RawMessage) error {
	// Parse created timestamp from event data
	var eventData map[string]interface{}
	if err := json.Unmarshal(data, &eventData); err != nil {
		r.logger.Warn("Failed to parse event data for timestamp",
			zap.String("event_id", eventID),
			zap.Error(err))
	}

	var stripeCreatedAt *time.Time
	if created, ok := eventData["created"].(float64); ok {
		t := time.Unix(int64(created), 0)
		stripeCreatedAt = &t
	}

	event := &model.StripeWebhookEvent{
		StripeEventID:   eventID,
		EventType:       eventType,
		Status:          model.WebhookStatusPending,
		Data:            model.JSONB(eventData),
		StripeCreatedAt: stripeCreatedAt,
	}

	// Use ON CONFLICT to handle duplicate events
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(event).Error

	if err != nil {
		r.logger.Error("Failed to save webhook event",
			zap.String("event_id", eventID),
			zap.String("event_type", eventType),
			zap.Error(err))
		return fmt.Errorf("failed to save webhook event: %w", err)
	}

	return nil
}

// GetEvent retrieves a webhook event by ID
func (r *webhookRepository) GetEvent(ctx context.Context, eventID string) (*model.StripeWebhookEvent, error) {
	var event model.StripeWebhookEvent

	err := r.db.WithContext(ctx).
		Where("stripe_event_id = ?", eventID).
		First(&event).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get webhook event",
			zap.String("event_id", eventID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get webhook event: %w", err)
	}

	return &event, nil
}

// MarkProcessed marks a webhook event as processed
func (r *webhookRepository) MarkProcessed(ctx context.Context, eventID string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&model.StripeWebhookEvent{}).
		Where("stripe_event_id = ?", eventID).
		Updates(map[string]interface{}{
			"status":       model.WebhookStatusCompleted,
			"processed_at": &now,
		})

	if result.Error != nil {
		r.logger.Error("Failed to mark webhook as processed",
			zap.String("event_id", eventID),
			zap.Error(result.Error))
		return fmt.Errorf("failed to mark webhook as processed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("webhook event not found: %s", eventID)
	}

	return nil
}

// MarkFailed marks a webhook event as failed
func (r *webhookRepository) MarkFailed(ctx context.Context, eventID string, err error) error {
	// Get current event to increment attempts
	var event model.StripeWebhookEvent
	if dbErr := r.db.WithContext(ctx).
		Where("stripe_event_id = ?", eventID).
		First(&event).Error; dbErr != nil {
		r.logger.Error("Failed to get webhook event for failure update",
			zap.String("event_id", eventID),
			zap.Error(dbErr))
		return fmt.Errorf("failed to get webhook event: %w", dbErr)
	}

	// Calculate next retry time with exponential backoff
	attempts := event.ProcessingAttempts + 1
	retryMinutes := 5 * (1 << attempts) // 5, 10, 20, 40, etc.
	if retryMinutes > 1440 {            // Cap at 24 hours
		retryMinutes = 1440
	}
	nextRetry := time.Now().Add(time.Duration(retryMinutes) * time.Minute)

	errorMsg := err.Error()

	result := r.db.WithContext(ctx).
		Model(&model.StripeWebhookEvent{}).
		Where("stripe_event_id = ?", eventID).
		Updates(map[string]interface{}{
			"status":              model.WebhookStatusFailed,
			"processing_attempts": attempts,
			"last_error":          &errorMsg,
			"next_retry_at":       &nextRetry,
		})

	if result.Error != nil {
		r.logger.Error("Failed to mark webhook as failed",
			zap.String("event_id", eventID),
			zap.Error(result.Error))
		return fmt.Errorf("failed to mark webhook as failed: %w", result.Error)
	}

	return nil
}

// GetPendingEvents retrieves pending webhook events for processing
func (r *webhookRepository) GetPendingEvents(ctx context.Context, limit int) ([]*model.StripeWebhookEvent, error) {
	var events []*model.StripeWebhookEvent

	query := r.db.WithContext(ctx).
		Where("status IN (?, ?) AND (next_retry_at IS NULL OR next_retry_at <= ?)",
			model.WebhookStatusPending,
			model.WebhookStatusFailed,
			time.Now()).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&events).Error
	if err != nil {
		r.logger.Error("Failed to get pending webhook events",
			zap.Error(err))
		return nil, fmt.Errorf("failed to get pending webhook events: %w", err)
	}

	return events, nil
}
