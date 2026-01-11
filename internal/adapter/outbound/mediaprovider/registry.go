package mediaprovider

import (
	"fmt"
	"sync"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// Registry manages media vendor adapters.
type Registry struct {
	mu       sync.RWMutex
	adapters map[model.MediaProviderType]outbound.MediaVendorAdapterPort
}

// NewRegistry creates a new adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[model.MediaProviderType]outbound.MediaVendorAdapterPort),
	}
}

// Register registers an adapter.
func (r *Registry) Register(adapter outbound.MediaVendorAdapterPort) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Type()] = adapter
}

// Get returns an adapter by provider type.
func (r *Registry) Get(providerType model.MediaProviderType) (outbound.MediaVendorAdapterPort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.adapters[providerType]
	if !ok {
		return nil, fmt.Errorf("no media adapter for provider type: %s", providerType)
	}
	return a, nil
}

// GetForProvider returns an adapter for a provider.
func (r *Registry) GetForProvider(prov *model.MediaProvider) (outbound.MediaVendorAdapterPort, error) {
	return r.Get(prov.Type)
}

// All returns all registered adapters.
func (r *Registry) All() []outbound.MediaVendorAdapterPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]outbound.MediaVendorAdapterPort, 0, len(r.adapters))
	for _, a := range r.adapters {
		result = append(result, a)
	}
	return result
}

// SupportedTypes returns all supported provider types.
func (r *Registry) SupportedTypes() []model.MediaProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.MediaProviderType, 0, len(r.adapters))
	for t := range r.adapters {
		result = append(result, t)
	}
	return result
}

// Compile-time interface check
var _ outbound.MediaVendorRegistryPort = (*Registry)(nil)
