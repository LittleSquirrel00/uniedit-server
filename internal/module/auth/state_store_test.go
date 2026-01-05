package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStateStore(t *testing.T) {
	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		store := NewMemoryStateStore(time.Minute)

		err := store.Set(ctx, "state-123", "github")
		require.NoError(t, err)

		data, err := store.Get(ctx, "state-123")
		require.NoError(t, err)
		assert.Equal(t, "github", data)
	})

	t.Run("Get returns error for non-existent state", func(t *testing.T) {
		store := NewMemoryStateStore(time.Minute)

		_, err := store.Get(ctx, "non-existent")
		assert.Error(t, err)
	})

	t.Run("Delete removes state", func(t *testing.T) {
		store := NewMemoryStateStore(time.Minute)

		err := store.Set(ctx, "state-123", "github")
		require.NoError(t, err)

		err = store.Delete(ctx, "state-123")
		require.NoError(t, err)

		_, err = store.Get(ctx, "state-123")
		assert.Error(t, err)
	})

	t.Run("Expired state returns error", func(t *testing.T) {
		store := NewMemoryStateStore(10 * time.Millisecond)

		err := store.Set(ctx, "state-123", "github")
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(20 * time.Millisecond)

		_, err = store.Get(ctx, "state-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("Default TTL is applied", func(t *testing.T) {
		store := NewMemoryStateStore(0) // Should use default 10 minutes
		assert.Equal(t, 10*time.Minute, store.ttl)
	})
}
