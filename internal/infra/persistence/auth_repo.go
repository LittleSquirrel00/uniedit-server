package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/infra/persistence/entity"
)

// RefreshTokenRepository implements auth.RefreshTokenRepository.
type RefreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new refresh token repository.
func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

var _ auth.RefreshTokenRepository = (*RefreshTokenRepository)(nil)

func (r *RefreshTokenRepository) Create(ctx context.Context, token *auth.RefreshToken) error {
	e := entity.FromDomainRefreshToken(token)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	var e entity.RefreshTokenEntity
	if err := r.db.WithContext(ctx).First(&e, "token_hash = ?", tokenHash).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, auth.ErrTokenNotFound
		}
		return nil, fmt.Errorf("get refresh token by hash: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.RefreshTokenEntity{}).
		Where("id = ?", id).
		Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("revoke refresh token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return auth.ErrTokenNotFound
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&entity.RefreshTokenEntity{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error; err != nil {
		return fmt.Errorf("revoke all tokens for user: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Delete(&entity.RefreshTokenEntity{}, "expires_at < ?", time.Now())
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// UserAPIKeyRepository implements auth.UserAPIKeyRepository.
type UserAPIKeyRepository struct {
	db *gorm.DB
}

// NewUserAPIKeyRepository creates a new user API key repository.
func NewUserAPIKeyRepository(db *gorm.DB) *UserAPIKeyRepository {
	return &UserAPIKeyRepository{db: db}
}

var _ auth.UserAPIKeyRepository = (*UserAPIKeyRepository)(nil)

func (r *UserAPIKeyRepository) Create(ctx context.Context, key *auth.UserAPIKey) error {
	e := entity.FromDomainUserAPIKey(key)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("create user API key: %w", err)
	}
	return nil
}

func (r *UserAPIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*auth.UserAPIKey, error) {
	var e entity.UserAPIKeyEntity
	if err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, auth.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("get user API key by ID: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *UserAPIKeyRepository) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*auth.UserAPIKey, error) {
	var e entity.UserAPIKeyEntity
	if err := r.db.WithContext(ctx).
		First(&e, "user_id = ? AND provider = ?", userID, provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, auth.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("get user API key by provider: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *UserAPIKeyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*auth.UserAPIKey, error) {
	var entities []entity.UserAPIKeyEntity
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("list user API keys: %w", err)
	}

	keys := make([]*auth.UserAPIKey, len(entities))
	for i, e := range entities {
		keys[i] = e.ToDomain()
	}
	return keys, nil
}

func (r *UserAPIKeyRepository) Update(ctx context.Context, key *auth.UserAPIKey) error {
	e := entity.FromDomainUserAPIKey(key)
	if err := r.db.WithContext(ctx).Save(e).Error; err != nil {
		return fmt.Errorf("update user API key: %w", err)
	}
	return nil
}

func (r *UserAPIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.UserAPIKeyEntity{}).
		Where("id = ?", id).
		Update("last_used_at", now)
	if result.Error != nil {
		return fmt.Errorf("update last used: %w", result.Error)
	}
	return nil
}

func (r *UserAPIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entity.UserAPIKeyEntity{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete user API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return auth.ErrAPIKeyNotFound
	}
	return nil
}

// SystemAPIKeyRepository implements auth.SystemAPIKeyRepository.
type SystemAPIKeyRepository struct {
	db *gorm.DB
}

// NewSystemAPIKeyRepository creates a new system API key repository.
func NewSystemAPIKeyRepository(db *gorm.DB) *SystemAPIKeyRepository {
	return &SystemAPIKeyRepository{db: db}
}

var _ auth.SystemAPIKeyRepository = (*SystemAPIKeyRepository)(nil)

func (r *SystemAPIKeyRepository) Create(ctx context.Context, key *auth.SystemAPIKey) error {
	e := entity.FromDomainSystemAPIKey(key)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("create system API key: %w", err)
	}
	return nil
}

func (r *SystemAPIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*auth.SystemAPIKey, error) {
	var e entity.SystemAPIKeyEntity
	if err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, auth.ErrSystemAPIKeyNotFound
		}
		return nil, fmt.Errorf("get system API key by ID: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *SystemAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*auth.SystemAPIKey, error) {
	var e entity.SystemAPIKeyEntity
	if err := r.db.WithContext(ctx).First(&e, "key_hash = ?", keyHash).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, auth.ErrSystemAPIKeyNotFound
		}
		return nil, fmt.Errorf("get system API key by hash: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *SystemAPIKeyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*auth.SystemAPIKey, error) {
	var entities []entity.SystemAPIKeyEntity
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("list system API keys: %w", err)
	}

	keys := make([]*auth.SystemAPIKey, len(entities))
	for i, e := range entities {
		keys[i] = e.ToDomain()
	}
	return keys, nil
}

func (r *SystemAPIKeyRepository) Update(ctx context.Context, key *auth.SystemAPIKey) error {
	e := entity.FromDomainSystemAPIKey(key)
	if err := r.db.WithContext(ctx).Save(e).Error; err != nil {
		return fmt.Errorf("update system API key: %w", err)
	}
	return nil
}

func (r *SystemAPIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.SystemAPIKeyEntity{}).
		Where("id = ?", id).
		Update("last_used_at", now)
	if result.Error != nil {
		return fmt.Errorf("update last used: %w", result.Error)
	}
	return nil
}

func (r *SystemAPIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entity.SystemAPIKeyEntity{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete system API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return auth.ErrSystemAPIKeyNotFound
	}
	return nil
}

func (r *SystemAPIKeyRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entity.SystemAPIKeyEntity{}).
		Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count system API keys: %w", err)
	}
	return count, nil
}
