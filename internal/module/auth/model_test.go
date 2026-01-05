package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOAuthProvider(t *testing.T) {
	t.Run("String returns correct value", func(t *testing.T) {
		assert.Equal(t, "github", OAuthProviderGitHub.String())
		assert.Equal(t, "google", OAuthProviderGoogle.String())
	})

	t.Run("IsValid returns true for valid providers", func(t *testing.T) {
		assert.True(t, OAuthProviderGitHub.IsValid())
		assert.True(t, OAuthProviderGoogle.IsValid())
	})

	t.Run("IsValid returns false for invalid providers", func(t *testing.T) {
		assert.False(t, OAuthProvider("invalid").IsValid())
		assert.False(t, OAuthProvider("").IsValid())
	})
}

func TestUserTableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName())
}

func TestRefreshToken(t *testing.T) {
	t.Run("TableName returns correct value", func(t *testing.T) {
		token := RefreshToken{}
		assert.Equal(t, "refresh_tokens", token.TableName())
	})

	t.Run("IsExpired returns true for expired token", func(t *testing.T) {
		token := &RefreshToken{
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		assert.True(t, token.IsExpired())
	})

	t.Run("IsExpired returns false for valid token", func(t *testing.T) {
		token := &RefreshToken{
			ExpiresAt: time.Now().Add(time.Hour),
		}
		assert.False(t, token.IsExpired())
	})

	t.Run("IsRevoked returns true for revoked token", func(t *testing.T) {
		now := time.Now()
		token := &RefreshToken{
			RevokedAt: &now,
		}
		assert.True(t, token.IsRevoked())
	})

	t.Run("IsRevoked returns false for non-revoked token", func(t *testing.T) {
		token := &RefreshToken{
			RevokedAt: nil,
		}
		assert.False(t, token.IsRevoked())
	})

	t.Run("IsValid returns true for valid non-expired non-revoked token", func(t *testing.T) {
		token := &RefreshToken{
			ExpiresAt: time.Now().Add(time.Hour),
			RevokedAt: nil,
		}
		assert.True(t, token.IsValid())
	})

	t.Run("IsValid returns false for expired token", func(t *testing.T) {
		token := &RefreshToken{
			ExpiresAt: time.Now().Add(-time.Hour),
			RevokedAt: nil,
		}
		assert.False(t, token.IsValid())
	})

	t.Run("IsValid returns false for revoked token", func(t *testing.T) {
		now := time.Now()
		token := &RefreshToken{
			ExpiresAt: time.Now().Add(time.Hour),
			RevokedAt: &now,
		}
		assert.False(t, token.IsValid())
	})
}

func TestUserAPIKey(t *testing.T) {
	t.Run("TableName returns correct value", func(t *testing.T) {
		key := UserAPIKey{}
		assert.Equal(t, "user_api_keys", key.TableName())
	})

	t.Run("HasScope returns true for existing scope", func(t *testing.T) {
		key := &UserAPIKey{
			Scopes: []string{"chat", "image"},
		}
		assert.True(t, key.HasScope(APIKeyScopeChat))
		assert.True(t, key.HasScope(APIKeyScopeImage))
	})

	t.Run("HasScope returns false for missing scope", func(t *testing.T) {
		key := &UserAPIKey{
			Scopes: []string{"chat"},
		}
		assert.False(t, key.HasScope(APIKeyScopeVideo))
		assert.False(t, key.HasScope(APIKeyScopeAudio))
	})

	t.Run("ToResponse converts correctly", func(t *testing.T) {
		now := time.Now()
		keyID := uuid.New()
		key := &UserAPIKey{
			ID:         keyID,
			Provider:   "openai",
			Name:       "My Key",
			KeyPrefix:  "sk-abc...",
			Scopes:     []string{"chat", "image"},
			LastUsedAt: &now,
			CreatedAt:  now,
		}

		resp := key.ToResponse()

		assert.Equal(t, keyID, resp.ID)
		assert.Equal(t, "openai", resp.Provider)
		assert.Equal(t, "My Key", resp.Name)
		assert.Equal(t, "sk-abc...", resp.KeyPrefix)
		assert.Equal(t, []string{"chat", "image"}, resp.Scopes)
		assert.Equal(t, &now, resp.LastUsedAt)
		assert.Equal(t, now, resp.CreatedAt)
	})
}

func TestUserToResponse(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	user := &User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "Test User",
		AvatarURL:     "https://example.com/avatar.png",
		OAuthProvider: OAuthProviderGitHub,
		CreatedAt:     now,
	}

	resp := user.ToResponse()

	assert.Equal(t, userID, resp.ID)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, "https://example.com/avatar.png", resp.AvatarURL)
	assert.Equal(t, "github", resp.Provider)
	assert.Equal(t, now, resp.CreatedAt)
}
