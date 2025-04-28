package entity

import (
	"encoding/json"
	"time"
)

// AuditLog 보안 추적 및 컴플라이언스를 위한 시스템 감사 이벤트를 저장합니다
type AuditLog struct {
	ID      uint
	UserID  *string
	Type    AuditLogType
	Content map[string]interface{}

	CreatedAt time.Time
}

// NewAuditLog 새 감사 로그 생성
func NewAuditLog(userID *string, logType AuditLogType, content map[string]interface{}) *AuditLog {
	return &AuditLog{
		UserID:    userID,
		Type:      logType,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// SetUserID 사용자 ID 설정
func (al *AuditLog) SetUserID(userID string) {
	al.UserID = &userID
}

// GetContent 콘텐츠 조회
func (al *AuditLog) GetContent() map[string]interface{} {
	return al.Content
}

// AddContentField 콘텐츠에 필드 추가
func (al *AuditLog) AddContentField(key string, value interface{}) {
	if al.Content == nil {
		al.Content = make(map[string]interface{})
	}
	al.Content[key] = value
}

// ContentJSON JSON 형식으로 콘텐츠 반환
func (al *AuditLog) ContentJSON() (string, error) {
	if al.Content == nil {
		return "{}", nil
	}

	data, err := json.Marshal(al.Content)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
