package outbound

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// RefreshTokenDatabasePort defines refresh token persistence operations.
type RefreshTokenDatabasePort interface {
	// Create creates a new refresh token.
	Create(ctx context.Context, token *model.RefreshToken) error

	// GetByHash gets a refresh token by its hash.
	GetByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)

	// Revoke revokes a refresh token.
	Revoke(ctx context.Context, id uuid.UUID) error

	// RevokeAllForUser revokes all refresh tokens for a user.
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error

	// DeleteExpired deletes expired refresh tokens.
	DeleteExpired(ctx context.Context) (int64, error)
}

// UserAPIKeyDatabasePort defines user API key persistence operations.
type UserAPIKeyDatabasePort interface {
	// Create creates a new API key.
	Create(ctx context.Context, key *model.UserAPIKey) error

	// GetByID gets an API key by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserAPIKey, error)

	// GetByUserAndProvider gets an API key by user and provider.
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*model.UserAPIKey, error)

	// ListByUser lists all API keys for a user.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error)

	// Update updates an API key.
	Update(ctx context.Context, key *model.UserAPIKey) error

	// UpdateLastUsed updates the last used timestamp.
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error

	// Delete deletes an API key.
	Delete(ctx context.Context, id uuid.UUID) error
}

// SystemAPIKeyDatabasePort defines system API key persistence operations.
type SystemAPIKeyDatabasePort interface {
	// Create creates a new system API key.
	Create(ctx context.Context, key *model.SystemAPIKey) error

	// GetByID gets a system API key by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.SystemAPIKey, error)

	// GetByHash gets a system API key by its hash.
	GetByHash(ctx context.Context, keyHash string) (*model.SystemAPIKey, error)

	// ListByUser lists all system API keys for a user.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.SystemAPIKey, error)

	// Update updates a system API key.
	Update(ctx context.Context, key *model.SystemAPIKey) error

	// UpdateLastUsed updates the last used timestamp.
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error

	// Delete deletes a system API key.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByUser counts system API keys for a user.
	CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)
}

// AuditLogDatabasePort defines audit log persistence operations.
type AuditLogDatabasePort interface {
	// Create creates a new audit log entry.
	Create(ctx context.Context, log *model.APIKeyAuditLog) error

	// ListByAPIKey lists audit logs for an API key.
	ListByAPIKey(ctx context.Context, apiKeyID uuid.UUID, limit int) ([]*model.APIKeyAuditLog, error)
}

// OAuthStateStorePort defines OAuth state storage operations.
type OAuthStateStorePort interface {
	// Set stores an OAuth state with provider info.
	Set(ctx context.Context, state string, provider string) error

	// Get retrieves the provider for a state.
	Get(ctx context.Context, state string) (string, error)

	// Delete removes a state.
	Delete(ctx context.Context, state string) error
}

// OAuthProviderPort defines OAuth provider operations.
type OAuthProviderPort interface {
	// GetAuthURL returns the OAuth authorization URL.
	GetAuthURL(state string) string

	// Exchange exchanges an authorization code for a token.
	Exchange(ctx context.Context, code string) (string, error)

	// GetUserInfo gets user information from the OAuth provider.
	GetUserInfo(ctx context.Context, token string) (*model.OAuthUserInfo, error)
}

// OAuthRegistryPort defines OAuth provider registry operations.
type OAuthRegistryPort interface {
	// Get returns an OAuth provider by name.
	Get(provider string) (OAuthProviderPort, error)

	// List returns all registered providers.
	List() []string
}

// CryptoPort defines encryption/decryption operations.
type CryptoPort interface {
	// Encrypt encrypts data.
	Encrypt(plaintext string) (string, error)

	// Decrypt decrypts data.
	Decrypt(ciphertext string) (string, error)
}

// JWTPort defines JWT token operations.
type JWTPort interface {
	// GenerateAccessToken generates an access token.
	GenerateAccessToken(userID uuid.UUID, email string) (string, time.Time, error)

	// GenerateRefreshToken generates a refresh token.
	GenerateRefreshToken() (rawToken string, tokenHash string, expiresAt time.Time, err error)

	// ValidateAccessToken validates an access token.
	ValidateAccessToken(token string) (*JWTClaims, error)

	// HashRefreshToken hashes a refresh token.
	HashRefreshToken(token string) string

	// GetAccessTokenExpiry returns access token expiry duration.
	GetAccessTokenExpiry() time.Duration

	// GetRefreshTokenExpiry returns refresh token expiry duration.
	GetRefreshTokenExpiry() time.Duration
}

// JWTClaims represents JWT token claims.
type JWTClaims struct {
	UserID uuid.UUID
	Email  string
}

// RateLimiterPort defines rate limiting operations.
type RateLimiterPort interface {
	// Allow checks if a request is allowed within rate limits.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)

	// AllowN checks if N requests are allowed.
	AllowN(ctx context.Context, key string, n int, limit int, window time.Duration) (bool, error)

	// GetRemaining returns remaining requests in window.
	GetRemaining(ctx context.Context, key string, limit int, window time.Duration) (int, error)
}
