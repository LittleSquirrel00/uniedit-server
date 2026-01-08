package auth

import (
	"time"

	"github.com/google/uuid"
)

// APIKeyScope represents the scope/permission of an API key.
type APIKeyScope string

const (
	APIKeyScopeChat      APIKeyScope = "chat"
	APIKeyScopeImage     APIKeyScope = "image"
	APIKeyScopeVideo     APIKeyScope = "video"
	APIKeyScopeAudio     APIKeyScope = "audio"
	APIKeyScopeEmbedding APIKeyScope = "embedding"
)

// UserAPIKey represents a user's stored API key for AI providers.
// This is for third-party provider keys (OpenAI, Anthropic, etc.) that users store.
type UserAPIKey struct {
	id           uuid.UUID
	userID       uuid.UUID
	provider     string   // openai, anthropic, etc.
	name         string   // user-defined name
	encryptedKey string   // AES-256-GCM encrypted
	keyPrefix    string   // first few chars (e.g., sk-abc)
	scopes       []string // permissions
	lastUsedAt   *time.Time
	createdAt    time.Time
	updatedAt    time.Time
}

// NewUserAPIKey creates a new user API key.
func NewUserAPIKey(
	userID uuid.UUID,
	provider, name, encryptedKey, keyPrefix string,
	scopes []string,
) *UserAPIKey {
	now := time.Now()
	return &UserAPIKey{
		id:           uuid.New(),
		userID:       userID,
		provider:     provider,
		name:         name,
		encryptedKey: encryptedKey,
		keyPrefix:    keyPrefix,
		scopes:       scopes,
		createdAt:    now,
		updatedAt:    now,
	}
}

// ReconstructUserAPIKey reconstructs a user API key from storage.
func ReconstructUserAPIKey(
	id, userID uuid.UUID,
	provider, name, encryptedKey, keyPrefix string,
	scopes []string,
	lastUsedAt *time.Time,
	createdAt, updatedAt time.Time,
) *UserAPIKey {
	return &UserAPIKey{
		id:           id,
		userID:       userID,
		provider:     provider,
		name:         name,
		encryptedKey: encryptedKey,
		keyPrefix:    keyPrefix,
		scopes:       scopes,
		lastUsedAt:   lastUsedAt,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// Getters

func (k *UserAPIKey) ID() uuid.UUID        { return k.id }
func (k *UserAPIKey) UserID() uuid.UUID    { return k.userID }
func (k *UserAPIKey) Provider() string     { return k.provider }
func (k *UserAPIKey) Name() string         { return k.name }
func (k *UserAPIKey) EncryptedKey() string { return k.encryptedKey }
func (k *UserAPIKey) KeyPrefix() string    { return k.keyPrefix }
func (k *UserAPIKey) Scopes() []string     { return k.scopes }
func (k *UserAPIKey) LastUsedAt() *time.Time { return k.lastUsedAt }
func (k *UserAPIKey) CreatedAt() time.Time { return k.createdAt }
func (k *UserAPIKey) UpdatedAt() time.Time { return k.updatedAt }

// Business logic

// HasScope checks if the API key has the specified scope.
func (k *UserAPIKey) HasScope(scope APIKeyScope) bool {
	for _, s := range k.scopes {
		if s == string(scope) {
			return true
		}
	}
	return false
}

// UpdateKey updates the encrypted key and prefix.
func (k *UserAPIKey) UpdateKey(encryptedKey, keyPrefix string) {
	k.encryptedKey = encryptedKey
	k.keyPrefix = keyPrefix
	k.updatedAt = time.Now()
}

// MarkUsed marks the key as used.
func (k *UserAPIKey) MarkUsed() {
	now := time.Now()
	k.lastUsedAt = &now
}

// BelongsTo checks if the key belongs to the specified user.
func (k *UserAPIKey) BelongsTo(userID uuid.UUID) bool {
	return k.userID == userID
}
