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

// refreshTokenAdapter implements outbound.RefreshTokenDatabasePort.
type refreshTokenAdapter struct {
	db *gorm.DB
}

// NewRefreshTokenAdapter creates a new refresh token database adapter.
func NewRefreshTokenAdapter(db *gorm.DB) outbound.RefreshTokenDatabasePort {
	return &refreshTokenAdapter{db: db}
}

func (a *refreshTokenAdapter) Create(ctx context.Context, token *model.RefreshToken) error {
	return a.db.WithContext(ctx).Create(token).Error
}

func (a *refreshTokenAdapter) GetByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := a.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auth.ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

func (a *refreshTokenAdapter) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := a.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrTokenNotFound
	}
	return nil
}

func (a *refreshTokenAdapter) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return a.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (a *refreshTokenAdapter) DeleteExpired(ctx context.Context) (int64, error) {
	result := a.db.WithContext(ctx).
		Delete(&model.RefreshToken{}, "expires_at < ?", time.Now())
	return result.RowsAffected, result.Error
}

// Compile-time check
var _ outbound.RefreshTokenDatabasePort = (*refreshTokenAdapter)(nil)
