package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByOAuth(ctx context.Context, provider OAuthProvider, oauthID string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// RefreshTokenRepository defines the interface for refresh token data access.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// APIKeyRepository defines the interface for API key data access.
type APIKeyRepository interface {
	Create(ctx context.Context, key *UserAPIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*UserAPIKey, error)
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*UserAPIKey, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*UserAPIKey, error)
	Update(ctx context.Context, key *UserAPIKey) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// --- User Repository Implementation ---

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByOAuth(ctx context.Context, provider OAuthProvider, oauthID string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "oauth_provider = ? AND oauth_id = ?", provider, oauthID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by oauth: %w", err)
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// --- Refresh Token Repository Implementation ---

type refreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new refresh token repository.
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *RefreshToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var token RefreshToken
	if err := r.db.WithContext(ctx).First(&token, "token_hash = ?", tokenHash).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &token, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&RefreshToken{}).Where("id = ?", id).Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("revoke refresh token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTokenNotFound
	}
	return nil
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error; err != nil {
		return fmt.Errorf("revoke all tokens for user: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Delete(&RefreshToken{}, "expires_at < ?", time.Now())
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// --- API Key Repository Implementation ---

type apiKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository creates a new API key repository.
func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, key *UserAPIKey) error {
	if err := r.db.WithContext(ctx).Create(key).Error; err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*UserAPIKey, error) {
	var key UserAPIKey
	if err := r.db.WithContext(ctx).First(&key, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("get api key: %w", err)
	}
	return &key, nil
}

func (r *apiKeyRepository) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*UserAPIKey, error) {
	var key UserAPIKey
	if err := r.db.WithContext(ctx).First(&key, "user_id = ? AND provider = ?", userID, provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("get api key by provider: %w", err)
	}
	return &key, nil
}

func (r *apiKeyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*UserAPIKey, error) {
	var keys []*UserAPIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	return keys, nil
}

func (r *apiKeyRepository) Update(ctx context.Context, key *UserAPIKey) error {
	if err := r.db.WithContext(ctx).Save(key).Error; err != nil {
		return fmt.Errorf("update api key: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&UserAPIKey{}).Where("id = ?", id).Update("last_used_at", now)
	if result.Error != nil {
		return fmt.Errorf("update last used: %w", result.Error)
	}
	return nil
}

func (r *apiKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&UserAPIKey{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete api key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

// --- System API Key Repository ---

// SystemAPIKeyRepository defines the interface for system API key data access.
type SystemAPIKeyRepository interface {
	Create(ctx context.Context, key *SystemAPIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*SystemAPIKey, error)
	GetByHash(ctx context.Context, keyHash string) (*SystemAPIKey, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*SystemAPIKey, error)
	Update(ctx context.Context, key *SystemAPIKey) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)
}

type systemAPIKeyRepository struct {
	db *gorm.DB
}

// NewSystemAPIKeyRepository creates a new system API key repository.
func NewSystemAPIKeyRepository(db *gorm.DB) SystemAPIKeyRepository {
	return &systemAPIKeyRepository{db: db}
}

func (r *systemAPIKeyRepository) Create(ctx context.Context, key *SystemAPIKey) error {
	if err := r.db.WithContext(ctx).Create(key).Error; err != nil {
		return fmt.Errorf("create system api key: %w", err)
	}
	return nil
}

func (r *systemAPIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*SystemAPIKey, error) {
	var key SystemAPIKey
	if err := r.db.WithContext(ctx).First(&key, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSystemAPIKeyNotFound
		}
		return nil, fmt.Errorf("get system api key: %w", err)
	}
	return &key, nil
}

func (r *systemAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*SystemAPIKey, error) {
	var key SystemAPIKey
	if err := r.db.WithContext(ctx).First(&key, "key_hash = ?", keyHash).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSystemAPIKeyNotFound
		}
		return nil, fmt.Errorf("get system api key by hash: %w", err)
	}
	return &key, nil
}

func (r *systemAPIKeyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*SystemAPIKey, error) {
	var keys []*SystemAPIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("list system api keys: %w", err)
	}
	return keys, nil
}

func (r *systemAPIKeyRepository) Update(ctx context.Context, key *SystemAPIKey) error {
	if err := r.db.WithContext(ctx).Save(key).Error; err != nil {
		return fmt.Errorf("update system api key: %w", err)
	}
	return nil
}

func (r *systemAPIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&SystemAPIKey{}).Where("id = ?", id).Update("last_used_at", now)
	if result.Error != nil {
		return fmt.Errorf("update last used: %w", result.Error)
	}
	return nil
}

func (r *systemAPIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&SystemAPIKey{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete system api key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSystemAPIKeyNotFound
	}
	return nil
}

func (r *systemAPIKeyRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&SystemAPIKey{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count system api keys: %w", err)
	}
	return count, nil
}
