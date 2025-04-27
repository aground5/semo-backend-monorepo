package entity

import (
	"errors"
	"time"
)

// NotificationType 알림 유형 정의
type NotificationType string

const (
	TypeEmail NotificationType = "email"
	TypeSMS   NotificationType = "sms"
	TypePush  NotificationType = "push"
)

// Notification 알림 엔티티
type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Content   string           `json:"content"`
	Read      bool             `json:"read"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// NewNotification 새 알림 생성
func NewNotification(userID, title, content string, notificationType NotificationType) (*Notification, error) {
	if userID == "" {
		return nil, errors.New("사용자 ID는 필수입니다")
	}

	if title == "" {
		return nil, errors.New("제목은 필수입니다")
	}

	now := time.Now()

	return &Notification{
		UserID:    userID,
		Type:      notificationType,
		Title:     title,
		Content:   content,
		Read:      false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
