// File: internal/logics/audit_log_service.go
package logics

import (
	"encoding/json"
	"fmt"

	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"go.uber.org/zap"
)

// AuditLogService provides methods for recording audit logs
type AuditLogService struct{}

// NewAuditLogService creates a new AuditLogService
func NewAuditLogService() *AuditLogService {
	return &AuditLogService{}
}

// AddLog adds a new audit log record to the audit_logs table.
// Parameters:
//   - logType: the type of the audit log (e.g. models.AuditLogTypeLoginSuccess)
//   - content: arbitrary key-value structured data that will be marshaled to JSON (stored as jsonb)
//   - userID: (optional) pointer to the user ID associated with this log; can be nil if not applicable.
func (s *AuditLogService) AddLog(logType models.AuditLogType, content interface{}, userID *string) error {
	// Marshal the content to JSON bytes.
	jsonData, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	// Create the audit log record.
	auditLog := models.AuditLog{
		UserID:  userID,
		Type:    logType,
		Content: jsonData, // datatypes.JSON is defined as []byte
	}

	// Insert the record into the database.
	if err := repositories.DBS.Postgres.Create(&auditLog).Error; err != nil {
		return fmt.Errorf("failed to insert audit log record: %w", err)
	}

	configs.Logger.Info("Audit log added", zap.String("type", string(logType)))
	return nil
}

// Global instance of AuditLogService
var AuditLogSvc = NewAuditLogService()
