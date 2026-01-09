package aiprovider

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// Registry manages vendor adapter instances.
type Registry struct {
	mu       sync.RWMutex
	adapters map[model.AIProviderType]outbound.AIVendorAdapterPort
}

// NewRegistry creates a new vendor adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[model.AIProviderType]outbound.AIVendorAdapterPort),
	}
}

// NewDefaultRegistry creates a registry with default adapters registered using the given HTTP client.
func NewDefaultRegistry(client *http.Client) outbound.AIVendorRegistryPort {
	r := NewRegistry()
	r.Register(NewOpenAIAdapter(client))
	r.Register(NewAnthropicAdapter(client))
	r.Register(NewGenericAdapter(client))
	return r
}

// Register registers an adapter.
func (r *Registry) Register(adapter outbound.AIVendorAdapterPort) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Type()] = adapter
}

// Get returns an adapter by provider type.
func (r *Registry) Get(providerType model.AIProviderType) (outbound.AIVendorAdapterPort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[providerType]
	if !ok {
		return nil, fmt.Errorf("adapter not found for type: %s", providerType)
	}
	return adapter, nil
}

// GetForProvider returns an adapter for a provider.
func (r *Registry) GetForProvider(provider *model.AIProvider) (outbound.AIVendorAdapterPort, error) {
	return r.Get(provider.Type)
}

// SupportedTypes returns all registered adapter types.
func (r *Registry) SupportedTypes() []model.AIProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]model.AIProviderType, 0, len(r.adapters))
	for t := range r.adapters {
		types = append(types, t)
	}
	return types
}


// Compile-time interface assertions
var _ outbound.AIVendorRegistryPort = (*Registry)(nil)
