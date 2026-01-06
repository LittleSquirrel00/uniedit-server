package adapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/uniedit/server/internal/module/ai/provider"
)

// Registry manages adapter instances.
type Registry struct {
	mu       sync.RWMutex
	adapters map[provider.ProviderType]Adapter
}

var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetRegistry returns the global adapter registry.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = NewRegistry()
		// Register default adapters
		globalRegistry.Register(NewOpenAIAdapter())
		globalRegistry.Register(NewAnthropicAdapter())
		globalRegistry.Register(NewGenericAdapter())
	})
	return globalRegistry
}

// NewRegistry creates a new adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[provider.ProviderType]Adapter),
	}
}

// Register registers an adapter.
func (r *Registry) Register(adapter Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Type()] = adapter
}

// Get returns an adapter by type.
func (r *Registry) Get(providerType provider.ProviderType) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[providerType]
	if !ok {
		return nil, fmt.Errorf("adapter not found for type: %s", providerType)
	}
	return adapter, nil
}

// GetForProvider returns an adapter for a provider.
func (r *Registry) GetForProvider(prov *provider.Provider) (Adapter, error) {
	return r.Get(prov.Type)
}

// Has checks if an adapter is registered.
func (r *Registry) Has(providerType provider.ProviderType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.adapters[providerType]
	return ok
}

// Types returns all registered adapter types.
func (r *Registry) Types() []provider.ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]provider.ProviderType, 0, len(r.adapters))
	for t := range r.adapters {
		types = append(types, t)
	}
	return types
}

// HealthCheck performs a health check on a provider using the appropriate adapter.
// This implements the provider.HealthChecker interface.
func (r *Registry) HealthCheck(ctx context.Context, prov *provider.Provider) error {
	adapter, err := r.Get(prov.Type)
	if err != nil {
		return fmt.Errorf("get adapter: %w", err)
	}
	return adapter.HealthCheck(ctx, prov)
}

// GetTextAdapter returns a TextAdapter for a provider type.
func (r *Registry) GetTextAdapter(providerType provider.ProviderType) (TextAdapter, error) {
	adapter, err := r.Get(providerType)
	if err != nil {
		return nil, err
	}
	return adapter, nil
}

// GetEmbeddingAdapter returns an EmbeddingAdapter for a provider type.
func (r *Registry) GetEmbeddingAdapter(providerType provider.ProviderType) (EmbeddingAdapter, error) {
	adapter, err := r.Get(providerType)
	if err != nil {
		return nil, err
	}
	return adapter, nil
}

// GetHealthChecker returns a HealthChecker for a provider type.
func (r *Registry) GetHealthChecker(providerType provider.ProviderType) (HealthChecker, error) {
	adapter, err := r.Get(providerType)
	if err != nil {
		return nil, err
	}
	return adapter, nil
}
