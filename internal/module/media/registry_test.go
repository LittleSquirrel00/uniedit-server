package media

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdapterRegistry_Register(t *testing.T) {
	registry := NewAdapterRegistry()

	// Create a mock adapter
	adapter := &mockAdapter{providerType: ProviderTypeOpenAI}

	registry.Register(adapter)

	result, err := registry.Get(ProviderTypeOpenAI)
	assert.NoError(t, err)
	assert.Equal(t, adapter, result)
}

func TestAdapterRegistry_Get_NotFound(t *testing.T) {
	registry := NewAdapterRegistry()

	_, err := registry.Get(ProviderTypeOpenAI)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no media adapter")
}

func TestAdapterRegistry_All(t *testing.T) {
	registry := NewAdapterRegistry()

	adapter1 := &mockAdapter{providerType: ProviderTypeOpenAI}
	adapter2 := &mockAdapter{providerType: ProviderTypeAnthropic}

	registry.Register(adapter1)
	registry.Register(adapter2)

	all := registry.All()
	assert.Len(t, all, 2)
}

func TestAdapterRegistry_SupportedTypes(t *testing.T) {
	registry := NewAdapterRegistry()

	adapter := &mockAdapter{providerType: ProviderTypeOpenAI}
	registry.Register(adapter)

	types := registry.SupportedTypes()
	assert.Len(t, types, 1)
	assert.Contains(t, types, ProviderTypeOpenAI)
}

func TestAdapterRegistry_GetForProvider(t *testing.T) {
	registry := NewAdapterRegistry()

	adapter := &mockAdapter{providerType: ProviderTypeOpenAI}
	registry.Register(adapter)

	provider := &Provider{Type: ProviderTypeOpenAI}
	result, err := registry.GetForProvider(provider)
	assert.NoError(t, err)
	assert.Equal(t, adapter, result)
}

// mockAdapter is a test adapter
type mockAdapter struct {
	providerType ProviderType
}

func (m *mockAdapter) Type() ProviderType {
	return m.providerType
}

func (m *mockAdapter) GenerateImage(_ context.Context, _ *ImageRequest, _ *Model, _ *Provider) (*ImageResponse, error) {
	return nil, nil
}

func (m *mockAdapter) GenerateVideo(_ context.Context, _ *VideoRequest, _ *Model, _ *Provider) (*VideoResponse, error) {
	return nil, nil
}

func (m *mockAdapter) GetVideoStatus(_ context.Context, _ string, _ *Provider) (*VideoStatus, error) {
	return nil, nil
}

func (m *mockAdapter) SupportsCapability(_ Capability) bool {
	return true
}

func (m *mockAdapter) HealthCheck(_ context.Context, _ *Provider) error {
	return nil
}
