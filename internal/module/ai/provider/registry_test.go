package provider

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	providers []*Provider
	err       error
}

func (m *MockRepository) CreateProvider(_ context.Context, _ *Provider) error {
	return m.err
}

func (m *MockRepository) GetProvider(_ context.Context, _ uuid.UUID) (*Provider, error) {
	return nil, m.err
}

func (m *MockRepository) GetProviderByName(_ context.Context, _ string) (*Provider, error) {
	return nil, m.err
}

func (m *MockRepository) ListProviders(_ context.Context, _ bool) ([]*Provider, error) {
	return m.providers, m.err
}

func (m *MockRepository) ListProvidersWithModels(_ context.Context, _ bool) ([]*Provider, error) {
	return m.providers, m.err
}

func (m *MockRepository) UpdateProvider(_ context.Context, _ *Provider) error {
	return m.err
}

func (m *MockRepository) DeleteProvider(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *MockRepository) CreateModel(_ context.Context, _ *Model) error {
	return m.err
}

func (m *MockRepository) GetModel(_ context.Context, _ string) (*Model, error) {
	return nil, m.err
}

func (m *MockRepository) ListModels(_ context.Context, _ *uuid.UUID, _ bool) ([]*Model, error) {
	return nil, m.err
}

func (m *MockRepository) ListModelsByCapability(_ context.Context, _ Capability) ([]*Model, error) {
	return nil, m.err
}

func (m *MockRepository) UpdateModel(_ context.Context, _ *Model) error {
	return m.err
}

func (m *MockRepository) DeleteModel(_ context.Context, _ string) error {
	return m.err
}

func (m *MockRepository) GetProviderWithModels(_ context.Context, _ uuid.UUID) (*Provider, error) {
	return nil, m.err
}

func newTestProviders() []*Provider {
	p1ID := uuid.New()
	p2ID := uuid.New()

	return []*Provider{
		{
			ID:      p1ID,
			Name:    "OpenAI",
			Type:    ProviderTypeOpenAI,
			Enabled: true,
			Models: []*Model{
				{
					ID:            "gpt-4o",
					ProviderID:    p1ID,
					Name:          "GPT-4o",
					Capabilities:  pq.StringArray{"chat", "vision", "tools"},
					ContextWindow: 128000,
					Enabled:       true,
				},
				{
					ID:            "gpt-4o-mini",
					ProviderID:    p1ID,
					Name:          "GPT-4o Mini",
					Capabilities:  pq.StringArray{"chat", "vision"},
					ContextWindow: 128000,
					Enabled:       true,
				},
				{
					ID:            "text-embedding-3-small",
					ProviderID:    p1ID,
					Name:          "Text Embedding 3 Small",
					Capabilities:  pq.StringArray{"embedding"},
					ContextWindow: 8191,
					Enabled:       true,
				},
			},
		},
		{
			ID:      p2ID,
			Name:    "Anthropic",
			Type:    ProviderTypeAnthropic,
			Enabled: true,
			Models: []*Model{
				{
					ID:            "claude-3-5-sonnet",
					ProviderID:    p2ID,
					Name:          "Claude 3.5 Sonnet",
					Capabilities:  pq.StringArray{"chat", "vision"},
					ContextWindow: 200000,
					Enabled:       true,
				},
			},
		},
	}
}

func TestDefaultRegistryConfig(t *testing.T) {
	config := DefaultRegistryConfig()
	assert.NotNil(t, config)
	assert.NotZero(t, config.RefreshInterval)
}

func TestNewRegistry(t *testing.T) {
	t.Run("Creates with default config", func(t *testing.T) {
		repo := &MockRepository{}
		registry := NewRegistry(repo, nil)

		assert.NotNil(t, registry)
		assert.NotNil(t, registry.providers)
		assert.NotNil(t, registry.models)
	})

	t.Run("Creates with custom config", func(t *testing.T) {
		repo := &MockRepository{}
		config := &RegistryConfig{RefreshInterval: 10}
		registry := NewRegistry(repo, config)

		assert.NotNil(t, registry)
		assert.Equal(t, config.RefreshInterval, registry.refreshInterval)
	})
}

func TestRegistry_Refresh(t *testing.T) {
	t.Run("Loads providers and models", func(t *testing.T) {
		providers := newTestProviders()
		repo := &MockRepository{providers: providers}
		registry := NewRegistry(repo, nil)

		err := registry.Refresh(context.Background())
		require.NoError(t, err)

		// Check providers loaded
		assert.Equal(t, 2, len(registry.AllProviders()))

		// Check models loaded
		assert.Equal(t, 4, len(registry.AllModels()))
	})

	t.Run("Returns error on repository failure", func(t *testing.T) {
		repo := &MockRepository{err: assert.AnError}
		registry := NewRegistry(repo, nil)

		err := registry.Refresh(context.Background())
		assert.Error(t, err)
	})
}

func TestRegistry_GetProvider(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	t.Run("Returns provider by ID", func(t *testing.T) {
		p, ok := registry.GetProvider(providers[0].ID)
		assert.True(t, ok)
		assert.Equal(t, providers[0].Name, p.Name)
	})

	t.Run("Returns false for non-existing ID", func(t *testing.T) {
		_, ok := registry.GetProvider(uuid.New())
		assert.False(t, ok)
	})
}

func TestRegistry_GetProvidersByType(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	t.Run("Returns providers by type", func(t *testing.T) {
		openaiProviders := registry.GetProvidersByType(ProviderTypeOpenAI)
		assert.Equal(t, 1, len(openaiProviders))
		assert.Equal(t, "OpenAI", openaiProviders[0].Name)
	})

	t.Run("Returns empty for non-existing type", func(t *testing.T) {
		ollamaProviders := registry.GetProvidersByType(ProviderTypeOllama)
		assert.Equal(t, 0, len(ollamaProviders))
	})
}

func TestRegistry_GetModel(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	t.Run("Returns model by ID", func(t *testing.T) {
		m, ok := registry.GetModel("gpt-4o")
		assert.True(t, ok)
		assert.Equal(t, "GPT-4o", m.Name)
	})

	t.Run("Returns false for non-existing model", func(t *testing.T) {
		_, ok := registry.GetModel("non-existing")
		assert.False(t, ok)
	})
}

func TestRegistry_GetModelWithProvider(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	t.Run("Returns model with provider", func(t *testing.T) {
		m, p, ok := registry.GetModelWithProvider("gpt-4o")
		assert.True(t, ok)
		assert.Equal(t, "GPT-4o", m.Name)
		assert.Equal(t, "OpenAI", p.Name)
	})

	t.Run("Returns false for non-existing model", func(t *testing.T) {
		_, _, ok := registry.GetModelWithProvider("non-existing")
		assert.False(t, ok)
	})
}

func TestRegistry_GetModelsByCapability(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	t.Run("Returns models with chat capability", func(t *testing.T) {
		models := registry.GetModelsByCapability(CapabilityChat)
		assert.Equal(t, 3, len(models))
	})

	t.Run("Returns models with vision capability", func(t *testing.T) {
		models := registry.GetModelsByCapability(CapabilityVision)
		assert.Equal(t, 3, len(models))
	})

	t.Run("Returns models with embedding capability", func(t *testing.T) {
		models := registry.GetModelsByCapability(CapabilityEmbedding)
		assert.Equal(t, 1, len(models))
	})

	t.Run("Returns empty for non-existing capability", func(t *testing.T) {
		models := registry.GetModelsByCapability(CapabilityVideo)
		assert.Equal(t, 0, len(models))
	})
}

func TestRegistry_GetModelsByCapabilities(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	t.Run("Returns all models for empty capabilities", func(t *testing.T) {
		models := registry.GetModelsByCapabilities([]Capability{})
		assert.Equal(t, 4, len(models))
	})

	t.Run("Returns models with all required capabilities", func(t *testing.T) {
		models := registry.GetModelsByCapabilities([]Capability{CapabilityChat, CapabilityVision})
		assert.Equal(t, 3, len(models))
	})

	t.Run("Returns models with tools capability", func(t *testing.T) {
		models := registry.GetModelsByCapabilities([]Capability{CapabilityChat, CapabilityTools})
		assert.Equal(t, 1, len(models))
	})

	t.Run("Returns empty for impossible combination", func(t *testing.T) {
		models := registry.GetModelsByCapabilities([]Capability{CapabilityEmbedding, CapabilityChat})
		assert.Equal(t, 0, len(models))
	})
}

func TestRegistry_GetModelsByProvider(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	t.Run("Returns models for provider", func(t *testing.T) {
		models := registry.GetModelsByProvider(providers[0].ID)
		assert.Equal(t, 3, len(models))
	})

	t.Run("Returns single model for Anthropic", func(t *testing.T) {
		models := registry.GetModelsByProvider(providers[1].ID)
		assert.Equal(t, 1, len(models))
	})

	t.Run("Returns empty for non-existing provider", func(t *testing.T) {
		models := registry.GetModelsByProvider(uuid.New())
		assert.Equal(t, 0, len(models))
	})
}

func TestRegistry_AllProviders(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	all := registry.AllProviders()
	assert.Equal(t, 2, len(all))
}

func TestRegistry_AllModels(t *testing.T) {
	providers := newTestProviders()
	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())

	all := registry.AllModels()
	assert.Equal(t, 4, len(all))
}
