package oauth

import (
	"fmt"
	"sync"

	"github.com/uniedit/server/internal/port/outbound"
)

// registry implements outbound.OAuthRegistryPort.
type registry struct {
	mu        sync.RWMutex
	providers map[string]outbound.OAuthProviderPort
}

// NewRegistry creates a new OAuth provider registry.
func NewRegistry() *registry {
	return &registry{
		providers: make(map[string]outbound.OAuthProviderPort),
	}
}

// Register registers an OAuth provider.
func (r *registry) Register(name string, provider outbound.OAuthProviderPort) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
}

// Get returns an OAuth provider by name.
func (r *registry) Get(name string) (outbound.OAuthProviderPort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// List returns all registered provider names.
func (r *registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Compile-time check
var _ outbound.OAuthRegistryPort = (*registry)(nil)
