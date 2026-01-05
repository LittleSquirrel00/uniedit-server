package oauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// MockProvider implements Provider for testing.
type MockProvider struct {
	name string
}

func (p *MockProvider) Name() string {
	return p.name
}

func (p *MockProvider) GetAuthURL(state string) string {
	return "https://example.com/auth?state=" + state
}

func (p *MockProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "mock-token"}, nil
}

func (p *MockProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	return &UserInfo{
		ID:    "123",
		Email: "test@example.com",
		Name:  "Test User",
	}, nil
}

func TestRegistry(t *testing.T) {
	t.Run("Register and Get", func(t *testing.T) {
		registry := NewRegistry()

		provider := &MockProvider{name: "github"}
		registry.Register(provider)

		retrieved, err := registry.Get("github")
		require.NoError(t, err)
		assert.Equal(t, "github", retrieved.Name())
	})

	t.Run("Get returns error for unregistered provider", func(t *testing.T) {
		registry := NewRegistry()

		_, err := registry.Get("unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("List returns all provider names", func(t *testing.T) {
		registry := NewRegistry()

		registry.Register(&MockProvider{name: "github"})
		registry.Register(&MockProvider{name: "google"})

		names := registry.List()
		assert.Len(t, names, 2)
		assert.Contains(t, names, "github")
		assert.Contains(t, names, "google")
	})

	t.Run("Has returns true for registered provider", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register(&MockProvider{name: "github"})

		assert.True(t, registry.Has("github"))
		assert.False(t, registry.Has("unknown"))
	})

	t.Run("Register overwrites existing provider", func(t *testing.T) {
		registry := NewRegistry()

		provider1 := &MockProvider{name: "github"}
		provider2 := &MockProvider{name: "github"}

		registry.Register(provider1)
		registry.Register(provider2)

		assert.Len(t, registry.List(), 1)
	})
}
