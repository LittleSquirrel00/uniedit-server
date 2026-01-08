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

// userAPIKeyAdapter implements outbound.UserAPIKeyDatabasePort.
type userAPIKeyAdapter struct {
	db *gorm.DB
}

// NewUserAPIKeyAdapter creates a new user API key database adapter.
func NewUserAPIKeyAdapter(db *gorm.DB) outbound.UserAPIKeyDatabasePort {
	return &userAPIKeyAdapter{db: db}
}

func (a *userAPIKeyAdapter) Create(ctx context.Context, key *model.UserAPIKey) error {
	return a.db.WithContext(ctx).Create(key).Error
}

func (a *userAPIKeyAdapter) GetByID(ctx context.Context, id uuid.UUID) (*model.UserAPIKey, error) {
	var key model.UserAPIKey
	err := a.db.WithContext(ctx).
		Where("id = ?", id).
		First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auth.ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (a *userAPIKeyAdapter) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*model.UserAPIKey, error) {
	var key model.UserAPIKey
	err := a.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auth.ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (a *userAPIKeyAdapter) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error) {
	var keys []*model.UserAPIKey
	err := a.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

func (a *userAPIKeyAdapter) Update(ctx context.Context, key *model.UserAPIKey) error {
	return a.db.WithContext(ctx).Save(key).Error
}

func (a *userAPIKeyAdapter) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return a.db.WithContext(ctx).
		Model(&model.UserAPIKey{}).
		Where("id = ?", id).
		Update("last_used_at", now).Error
}

func (a *userAPIKeyAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	result := a.db.WithContext(ctx).
		Delete(&model.UserAPIKey{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrAPIKeyNotFound
	}
	return nil
}

// Compile-time check
var _ outbound.UserAPIKeyDatabasePort = (*userAPIKeyAdapter)(nil)
