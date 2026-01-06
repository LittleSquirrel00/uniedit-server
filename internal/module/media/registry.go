package media

import (
	"fmt"
	"sync"
)

// AdapterRegistry manages media adapters.
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[ProviderType]Adapter
}

// NewAdapterRegistry creates a new adapter registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[ProviderType]Adapter),
	}
}

// Register registers an adapter.
func (r *AdapterRegistry) Register(a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[a.Type()] = a
}

// Get returns an adapter by provider type.
func (r *AdapterRegistry) Get(providerType ProviderType) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.adapters[providerType]
	if !ok {
		return nil, fmt.Errorf("no media adapter for provider type: %s", providerType)
	}
	return a, nil
}

// GetForProvider returns an adapter for a provider.
func (r *AdapterRegistry) GetForProvider(prov *Provider) (Adapter, error) {
	return r.Get(prov.Type)
}

// All returns all registered adapters.
func (r *AdapterRegistry) All() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		result = append(result, a)
	}
	return result
}

// SupportedTypes returns all supported provider types.
func (r *AdapterRegistry) SupportedTypes() []ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ProviderType, 0, len(r.adapters))
	for t := range r.adapters {
		result = append(result, t)
	}
	return result
}
