package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// systemAPIKeyAdapter implements outbound.SystemAPIKeyDatabasePort.
type systemAPIKeyAdapter struct {
	db *gorm.DB
}

// NewSystemAPIKeyAdapter creates a new system API key database adapter.
func NewSystemAPIKeyAdapter(db *gorm.DB) outbound.SystemAPIKeyDatabasePort {
	return &systemAPIKeyAdapter{db: db}
}

func (a *systemAPIKeyAdapter) Create(ctx context.Context, key *model.SystemAPIKey) error {
	return a.db.WithContext(ctx).Create(key).Error
}

func (a *systemAPIKeyAdapter) GetByID(ctx context.Context, id uuid.UUID) (*model.SystemAPIKey, error) {
	var key model.SystemAPIKey
	err := a.db.WithContext(ctx).
		Where("id = ?", id).
		First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auth.ErrSystemAPIKeyNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (a *systemAPIKeyAdapter) GetByHash(ctx context.Context, keyHash string) (*model.SystemAPIKey, error) {
	var key model.SystemAPIKey
	err := a.db.WithContext(ctx).
		Where("key_hash = ?", keyHash).
		First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auth.ErrSystemAPIKeyNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (a *systemAPIKeyAdapter) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.SystemAPIKey, error) {
	var keys []*model.SystemAPIKey
	err := a.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

func (a *systemAPIKeyAdapter) Update(ctx context.Context, key *model.SystemAPIKey) error {
	return a.db.WithContext(ctx).Save(key).Error
}

func (a *systemAPIKeyAdapter) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return a.db.WithContext(ctx).
		Model(&model.SystemAPIKey{}).
		Where("id = ?", id).
		Update("last_used_at", now).Error
}

func (a *systemAPIKeyAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	result := a.db.WithContext(ctx).
		Delete(&model.SystemAPIKey{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrSystemAPIKeyNotFound
	}
	return nil
}

func (a *systemAPIKeyAdapter) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := a.db.WithContext(ctx).
		Model(&model.SystemAPIKey{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// Compile-time check
var _ outbound.SystemAPIKeyDatabasePort = (*systemAPIKeyAdapter)(nil)
