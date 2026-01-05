package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig()
	assert.Equal(t, 15*time.Minute, config.AccessTokenExpiry)
	assert.Equal(t, 7*24*time.Hour, config.RefreshTokenExpiry)
	assert.Equal(t, "uniedit", config.Issuer)
}

func TestNewJWTManager(t *testing.T) {
	t.Run("creates with custom config", func(t *testing.T) {
		config := &JWTConfig{
			Secret:             "test-secret-key-that-is-long-enough",
			AccessTokenExpiry:  30 * time.Minute,
			RefreshTokenExpiry: 24 * time.Hour,
			Issuer:             "custom-issuer",
		}
		manager := NewJWTManager(config)
		assert.NotNil(t, manager)
		assert.Equal(t, 30*time.Minute, manager.GetAccessTokenExpiry())
		assert.Equal(t, 24*time.Hour, manager.GetRefreshTokenExpiry())
	})

	t.Run("creates with nil config uses defaults", func(t *testing.T) {
		manager := NewJWTManager(nil)
		assert.NotNil(t, manager)
		assert.Equal(t, 15*time.Minute, manager.GetAccessTokenExpiry())
	})
}

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	config := &JWTConfig{
		Secret:            "test-secret-key-that-is-long-enough",
		AccessTokenExpiry: 15 * time.Minute,
		Issuer:            "test",
	}
	manager := NewJWTManager(config)

	user := &User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	token, expiresAt, err := manager.GenerateAccessToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(16*time.Minute)))
}

func TestJWTManager_ValidateAccessToken(t *testing.T) {
	config := &JWTConfig{
		Secret:            "test-secret-key-that-is-long-enough",
		AccessTokenExpiry: 15 * time.Minute,
		Issuer:            "test",
	}
	manager := NewJWTManager(config)

	user := &User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	t.Run("validates valid token", func(t *testing.T) {
		token, _, err := manager.GenerateAccessToken(user)
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		_, err := manager.ValidateAccessToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("rejects token with wrong secret", func(t *testing.T) {
		token, _, err := manager.GenerateAccessToken(user)
		require.NoError(t, err)

		// Create manager with different secret
		otherManager := NewJWTManager(&JWTConfig{
			Secret: "different-secret-key-that-is-also-long",
		})

		_, err = otherManager.ValidateAccessToken(token)
		assert.Error(t, err)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		expiredConfig := &JWTConfig{
			Secret:            "test-secret-key-that-is-long-enough",
			AccessTokenExpiry: -time.Hour, // Already expired
			Issuer:            "test",
		}
		expiredManager := NewJWTManager(expiredConfig)

		token, _, err := expiredManager.GenerateAccessToken(user)
		require.NoError(t, err)

		_, err = expiredManager.ValidateAccessToken(token)
		assert.Error(t, err)
	})
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	config := &JWTConfig{
		Secret:             "test-secret-key-that-is-long-enough",
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager := NewJWTManager(config)

	rawToken, tokenHash, expiresAt, err := manager.GenerateRefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, rawToken)
	assert.NotEmpty(t, tokenHash)
	assert.Len(t, tokenHash, 64) // SHA-256 hex is 64 chars
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(8*24*time.Hour)))
}

func TestJWTManager_HashRefreshToken(t *testing.T) {
	manager := NewJWTManager(nil)

	t.Run("produces consistent hash", func(t *testing.T) {
		token := "test-token-123"
		hash1 := manager.HashRefreshToken(token)
		hash2 := manager.HashRefreshToken(token)
		assert.Equal(t, hash1, hash2)
		assert.Len(t, hash1, 64)
	})

	t.Run("produces different hash for different tokens", func(t *testing.T) {
		hash1 := manager.HashRefreshToken("token-1")
		hash2 := manager.HashRefreshToken("token-2")
		assert.NotEqual(t, hash1, hash2)
	})
}
