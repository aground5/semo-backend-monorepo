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

// TossWebhookRepository handles Toss webhook event storage and processing
type TossWebhookRepository interface {
	SaveEvent(ctx context.Context, eventID, eventType string, data json.RawMessage, metadata TossWebhookMetadata) error
	GetEvent(ctx context.Context, eventID string) (*model.TossWebhookEvent, error)
	MarkProcessed(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string, err error) error
	GetPendingEvents(ctx context.Context, limit int) ([]*model.TossWebhookEvent, error)
}

// TossWebhookMetadata contains additional metadata for Toss webhook events
type TossWebhookMetadata struct {
	EventStatus    *string
	PaymentKey     *string
	OrderID        *string
	TransactionKey *string
	Secret         *string
	IPAddress      *string
	UserAgent      *string
	TossCreatedAt  time.Time
}

type tossWebhookRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewTossWebhookRepository creates a new Toss webhook repository
func NewTossWebhookRepository(db *gorm.DB, logger *zap.Logger) TossWebhookRepository {
	return &tossWebhookRepository{
		db:     db,
		logger: logger,
	}
}

// SaveEvent saves a new Toss webhook event
func (r *tossWebhookRepository) SaveEvent(ctx context.Context, eventID, eventType string, data json.RawMessage, metadata TossWebhookMetadata) error {
	// Parse event data for validation
	var eventData map[string]interface{}
	if err := json.Unmarshal(data, &eventData); err != nil {
		r.logger.Warn("Failed to parse event data",
			zap.String("event_id", eventID),
			zap.Error(err))
	}

	event := &model.TossWebhookEvent{
		TossEventID:      eventID,
		EventType:        eventType,
		EventStatus:      metadata.EventStatus,
		ProcessingStatus: model.WebhookStatusPending,
		PaymentKey:       metadata.PaymentKey,
		OrderID:          metadata.OrderID,
		TransactionKey:   metadata.TransactionKey,
		Secret:           metadata.Secret,
		EventData:        model.JSONB(eventData),
		IPAddress:        metadata.IPAddress,
		UserAgent:        metadata.UserAgent,
		TossCreatedAt:    metadata.TossCreatedAt,
	}

	// Use ON CONFLICT to handle duplicate events
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(event).Error

	if err != nil {
		r.logger.Error("Failed to save Toss webhook event",
			zap.String("event_id", eventID),
			zap.String("event_type", eventType),
			zap.Error(err))
		return fmt.Errorf("failed to save Toss webhook event: %w", err)
	}

	return nil
}

// GetEvent retrieves a Toss webhook event by ID
func (r *tossWebhookRepository) GetEvent(ctx context.Context, eventID string) (*model.TossWebhookEvent, error) {
	var event model.TossWebhookEvent

	err := r.db.WithContext(ctx).
		Where("toss_event_id = ?", eventID).
		First(&event).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get Toss webhook event",
			zap.String("event_id", eventID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get Toss webhook event: %w", err)
	}

	return &event, nil
}

// MarkProcessed marks a Toss webhook event as processed
func (r *tossWebhookRepository) MarkProcessed(ctx context.Context, eventID string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&model.TossWebhookEvent{}).
		Where("toss_event_id = ?", eventID).
		Updates(map[string]interface{}{
			"processing_status": model.WebhookStatusCompleted,
			"processed_at":      &now,
		})

	if result.Error != nil {
		r.logger.Error("Failed to mark Toss webhook as processed",
			zap.String("event_id", eventID),
			zap.Error(result.Error))
		return fmt.Errorf("failed to mark Toss webhook as processed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("Toss webhook event not found: %s", eventID)
	}

	return nil
}

// MarkFailed marks a Toss webhook event as failed
func (r *tossWebhookRepository) MarkFailed(ctx context.Context, eventID string, err error) error {
	// Get current event to increment retry count
	var event model.TossWebhookEvent
	if dbErr := r.db.WithContext(ctx).
		Where("toss_event_id = ?", eventID).
		First(&event).Error; dbErr != nil {
		r.logger.Error("Failed to get Toss webhook event for failure update",
			zap.String("event_id", eventID),
			zap.Error(dbErr))
		return fmt.Errorf("failed to get Toss webhook event: %w", dbErr)
	}

	// Calculate next retry time with exponential backoff
	retryCount := event.RetryCount + 1
	retryMinutes := 5 * (1 << retryCount) // 5, 10, 20, 40, etc.
	if retryMinutes > 1440 {              // Cap at 24 hours
		retryMinutes = 1440
	}
	nextRetry := time.Now().Add(time.Duration(retryMinutes) * time.Minute)

	errorMsg := err.Error()

	result := r.db.WithContext(ctx).
		Model(&model.TossWebhookEvent{}).
		Where("toss_event_id = ?", eventID).
		Updates(map[string]interface{}{
			"processing_status": model.WebhookStatusFailed,
			"retry_count":       retryCount,
			"last_error":        &errorMsg,
			"next_retry_at":     &nextRetry,
		})

	if result.Error != nil {
		r.logger.Error("Failed to mark Toss webhook as failed",
			zap.String("event_id", eventID),
			zap.Error(result.Error))
		return fmt.Errorf("failed to mark Toss webhook as failed: %w", result.Error)
	}

	return nil
}

// GetPendingEvents retrieves pending Toss webhook events for processing
func (r *tossWebhookRepository) GetPendingEvents(ctx context.Context, limit int) ([]*model.TossWebhookEvent, error) {
	var events []*model.TossWebhookEvent

	query := r.db.WithContext(ctx).
		Where("processing_status IN (?, ?) AND (next_retry_at IS NULL OR next_retry_at <= ?)",
			model.WebhookStatusPending,
			model.WebhookStatusFailed,
			time.Now()).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&events).Error
	if err != nil {
		r.logger.Error("Failed to get pending Toss webhook events",
			zap.Error(err))
		return nil, fmt.Errorf("failed to get pending Toss webhook events: %w", err)
	}

	return events, nil
}