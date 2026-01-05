package media

import (
	"fmt"
	"sync"

	"github.com/uniedit/server/internal/module/ai/provider"
)

// Registry manages media adapters.
type Registry struct {
	mu       sync.RWMutex
	adapters map[provider.ProviderType]Adapter
}

var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetRegistry returns the global media adapter registry.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			adapters: make(map[provider.ProviderType]Adapter),
		}
		// Register default adapters
		globalRegistry.Register(NewOpenAIImageAdapter())
	})
	return globalRegistry
}

// Register registers an adapter.
func (r *Registry) Register(adapter Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Type()] = adapter
}

// Get returns an adapter by provider type.
func (r *Registry) Get(providerType provider.ProviderType) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[providerType]
	if !ok {
		return nil, fmt.Errorf("no media adapter for provider type: %s", providerType)
	}
	return adapter, nil
}

// GetForProvider returns an adapter for a provider.
func (r *Registry) GetForProvider(prov *provider.Provider) (Adapter, error) {
	return r.Get(prov.Type)
}

// All returns all registered adapters.
func (r *Registry) All() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Adapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		result = append(result, adapter)
	}
	return result
}

// SupportedTypes returns all supported provider types.
func (r *Registry) SupportedTypes() []provider.ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]provider.ProviderType, 0, len(r.adapters))
	for t := range r.adapters {
		result = append(result, t)
	}
	return result
}
