package model

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UniversalID *uuid.UUID `gorm:"column:universal_id;type:uuid;index" json:"universal_id,omitempty"`
	Action      string     `gorm:"not null;size:100" json:"action"`
	Table     string     `gorm:"column:table_name;not null;size:100;index:idx_audit_log_table_action" json:"table_name"`
	RecordID  *int64     `json:"record_id,omitempty"`
	OldValues JSONB      `gorm:"type:jsonb" json:"old_values,omitempty"`
	NewValues JSONB      `gorm:"type:jsonb" json:"new_values,omitempty"`
	IPAddress *net.IP    `gorm:"type:inet" json:"ip_address,omitempty"`
	Metadata  JSONB      `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt time.Time  `gorm:"default:now();index" json:"created_at"`
}

// TableName specifies the table name for GORM
func (AuditLog) TableName() string {
	return "audit_log"
}
