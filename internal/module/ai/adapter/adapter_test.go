package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uniedit/server/internal/module/ai/provider"
)

func TestBaseAdapter_SupportsCapability(t *testing.T) {
	adapter := NewBaseAdapter(
		provider.CapabilityChat,
		provider.CapabilityStream,
	)

	assert.True(t, adapter.SupportsCapability(provider.CapabilityChat))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityStream))
	assert.False(t, adapter.SupportsCapability(provider.CapabilityVision))
}

func TestBaseAdapter_AddCapability(t *testing.T) {
	adapter := NewBaseAdapter()

	assert.False(t, adapter.SupportsCapability(provider.CapabilityChat))

	adapter.AddCapability(provider.CapabilityChat)

	assert.True(t, adapter.SupportsCapability(provider.CapabilityChat))
}

func TestBaseAdapter_RemoveCapability(t *testing.T) {
	adapter := NewBaseAdapter(provider.CapabilityChat)

	assert.True(t, adapter.SupportsCapability(provider.CapabilityChat))

	adapter.RemoveCapability(provider.CapabilityChat)

	assert.False(t, adapter.SupportsCapability(provider.CapabilityChat))
}

func TestBaseAdapter_Capabilities(t *testing.T) {
	adapter := NewBaseAdapter(
		provider.CapabilityChat,
		provider.CapabilityStream,
		provider.CapabilityVision,
	)

	caps := adapter.Capabilities()
	assert.Len(t, caps, 3)
	assert.Contains(t, caps, provider.CapabilityChat)
	assert.Contains(t, caps, provider.CapabilityStream)
	assert.Contains(t, caps, provider.CapabilityVision)
}

func TestOpenAIAdapter_Type(t *testing.T) {
	adapter := NewOpenAIAdapter()
	assert.Equal(t, provider.ProviderTypeOpenAI, adapter.Type())
}

func TestOpenAIAdapter_Capabilities(t *testing.T) {
	adapter := NewOpenAIAdapter()

	assert.True(t, adapter.SupportsCapability(provider.CapabilityChat))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityStream))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityVision))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityTools))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityJSON))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityEmbedding))
	assert.False(t, adapter.SupportsCapability(provider.CapabilityImage))
}

func TestAnthropicAdapter_Type(t *testing.T) {
	adapter := NewAnthropicAdapter()
	assert.Equal(t, provider.ProviderTypeAnthropic, adapter.Type())
}

func TestAnthropicAdapter_Capabilities(t *testing.T) {
	adapter := NewAnthropicAdapter()

	assert.True(t, adapter.SupportsCapability(provider.CapabilityChat))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityStream))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityVision))
	assert.True(t, adapter.SupportsCapability(provider.CapabilityTools))
	assert.False(t, adapter.SupportsCapability(provider.CapabilityEmbedding))
}

func TestGenericAdapter_Type(t *testing.T) {
	adapter := NewGenericAdapter()
	assert.Equal(t, provider.ProviderTypeGeneric, adapter.Type())
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	adapter := NewOpenAIAdapter()
	registry.Register(adapter)

	result, err := registry.Get(provider.ProviderTypeOpenAI)
	assert.NoError(t, err)
	assert.Equal(t, adapter, result)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get(provider.ProviderTypeOpenAI)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "adapter not found")
}

func TestRegistry_Has(t *testing.T) {
	registry := NewRegistry()

	assert.False(t, registry.Has(provider.ProviderTypeOpenAI))

	registry.Register(NewOpenAIAdapter())

	assert.True(t, registry.Has(provider.ProviderTypeOpenAI))
}

func TestRegistry_Types(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewOpenAIAdapter())
	registry.Register(NewAnthropicAdapter())

	types := registry.Types()
	assert.Len(t, types, 2)
	assert.Contains(t, types, provider.ProviderTypeOpenAI)
	assert.Contains(t, types, provider.ProviderTypeAnthropic)
}

func TestRegistry_GetTextAdapter(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewOpenAIAdapter())

	adapter, err := registry.GetTextAdapter(provider.ProviderTypeOpenAI)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
}

func TestRegistry_GetEmbeddingAdapter(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewOpenAIAdapter())

	adapter, err := registry.GetEmbeddingAdapter(provider.ProviderTypeOpenAI)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
}

func TestRegistry_GetHealthChecker(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewOpenAIAdapter())

	checker, err := registry.GetHealthChecker(provider.ProviderTypeOpenAI)
	assert.NoError(t, err)
	assert.NotNil(t, checker)
}

// Test interface compliance
func TestInterfaceCompliance(t *testing.T) {
	// These should compile - asserting interface compliance
	var _ Adapter = (*OpenAIAdapter)(nil)
	var _ Adapter = (*AnthropicAdapter)(nil)
	var _ Adapter = (*GenericAdapter)(nil)

	var _ TextAdapter = (*OpenAIAdapter)(nil)
	var _ TextAdapter = (*AnthropicAdapter)(nil)
	var _ TextAdapter = (*GenericAdapter)(nil)

	var _ EmbeddingAdapter = (*OpenAIAdapter)(nil)
	var _ EmbeddingAdapter = (*AnthropicAdapter)(nil)
	var _ EmbeddingAdapter = (*GenericAdapter)(nil)

	var _ HealthChecker = (*OpenAIAdapter)(nil)
	var _ HealthChecker = (*AnthropicAdapter)(nil)
	var _ HealthChecker = (*GenericAdapter)(nil)

	var _ CapabilityChecker = (*BaseAdapter)(nil)
}
