package auth

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLogger handles API key audit logging.
type AuditLogger interface {
	// Log records an audit event.
	Log(ctx context.Context, keyID uuid.UUID, action APIKeyAuditAction, details map[string]any, ip, userAgent string) error
	// GetLogs retrieves audit logs for a key.
	GetLogs(ctx context.Context, keyID uuid.UUID, limit, offset int) ([]*APIKeyAuditLog, int64, error)
}

type auditLogger struct {
	db *gorm.DB
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(db *gorm.DB) AuditLogger {
	return &auditLogger{db: db}
}

func (l *auditLogger) Log(ctx context.Context, keyID uuid.UUID, action APIKeyAuditAction, details map[string]any, ip, userAgent string) error {
	log := &APIKeyAuditLog{
		APIKeyID:  keyID,
		Action:    action,
		Details:   details,
		IPAddress: ip,
		UserAgent: userAgent,
	}
	return l.db.WithContext(ctx).Create(log).Error
}

func (l *auditLogger) GetLogs(ctx context.Context, keyID uuid.UUID, limit, offset int) ([]*APIKeyAuditLog, int64, error) {
	var logs []*APIKeyAuditLog
	var total int64

	// Count total
	if err := l.db.WithContext(ctx).Model(&APIKeyAuditLog{}).Where("api_key_id = ?", keyID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get logs with pagination
	if err := l.db.WithContext(ctx).
		Where("api_key_id = ?", keyID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
