package auth

import (
	"context"

	"github.com/google/uuid"
)

// RefreshTokenRepository defines the interface for refresh token data access.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// UserAPIKeyRepository defines the interface for user API key data access.
type UserAPIKeyRepository interface {
	Create(ctx context.Context, key *UserAPIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*UserAPIKey, error)
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*UserAPIKey, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*UserAPIKey, error)
	Update(ctx context.Context, key *UserAPIKey) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

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

// OAuthStateStore defines the interface for OAuth state management.
type OAuthStateStore interface {
	Set(ctx context.Context, state string, provider OAuthProvider) error
	Get(ctx context.Context, state string) (OAuthProvider, error)
	Delete(ctx context.Context, state string) error
}
