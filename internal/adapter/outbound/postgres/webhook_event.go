package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// webhookEventAdapter implements outbound.WebhookEventDatabasePort.
type webhookEventAdapter struct {
	db *gorm.DB
}

// NewWebhookEventAdapter creates a new webhook event database adapter.
func NewWebhookEventAdapter(db *gorm.DB) outbound.WebhookEventDatabasePort {
	return &webhookEventAdapter{db: db}
}

func (a *webhookEventAdapter) Create(ctx context.Context, event *model.WebhookEvent) error {
	if err := a.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create webhook event: %w", err)
	}
	return nil
}

func (a *webhookEventAdapter) Exists(ctx context.Context, provider, eventID string) (bool, error) {
	var count int64
	err := a.db.WithContext(ctx).
		Model(&model.WebhookEvent{}).
		Where("provider = ? AND event_id = ?", provider, eventID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check webhook event exists: %w", err)
	}
	return count > 0, nil
}

func (a *webhookEventAdapter) MarkProcessed(ctx context.Context, id uuid.UUID, processErr error) error {
	now := time.Now()
	updates := map[string]interface{}{
		"processed":    true,
		"processed_at": now,
	}
	if processErr != nil {
		errStr := processErr.Error()
		updates["error"] = errStr
	}
	err := a.db.WithContext(ctx).
		Model(&model.WebhookEvent{}).
		Where("id = ?", id).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("mark webhook event processed: %w", err)
	}
	return nil
}

// Compile-time check
var _ outbound.WebhookEventDatabasePort = (*webhookEventAdapter)(nil)
