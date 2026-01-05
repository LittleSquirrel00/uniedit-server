package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Registry provides in-memory access to providers and models.
type Registry struct {
	mu sync.RWMutex

	// Indexed data
	providers      map[uuid.UUID]*Provider
	providersByType map[ProviderType][]*Provider
	models          map[string]*Model
	modelsByCapability map[Capability][]*Model
	modelsByProvider map[uuid.UUID][]*Model

	// Repository for database access
	repo Repository

	// Refresh configuration
	refreshInterval time.Duration
	stopRefresh     chan struct{}
}

// RegistryConfig contains registry configuration.
type RegistryConfig struct {
	RefreshInterval time.Duration
}

// DefaultRegistryConfig returns the default registry configuration.
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		RefreshInterval: 5 * time.Minute,
	}
}

// NewRegistry creates a new provider registry.
func NewRegistry(repo Repository, config *RegistryConfig) *Registry {
	if config == nil {
		config = DefaultRegistryConfig()
	}

	r := &Registry{
		providers:        make(map[uuid.UUID]*Provider),
		providersByType:  make(map[ProviderType][]*Provider),
		models:           make(map[string]*Model),
		modelsByCapability: make(map[Capability][]*Model),
		modelsByProvider:   make(map[uuid.UUID][]*Model),
		repo:             repo,
		refreshInterval:  config.RefreshInterval,
		stopRefresh:      make(chan struct{}),
	}

	return r
}

// Start starts the registry and loads initial data.
func (r *Registry) Start(ctx context.Context) error {
	if err := r.Refresh(ctx); err != nil {
		return fmt.Errorf("initial refresh: %w", err)
	}

	// Start background refresh
	go r.refreshLoop()

	return nil
}

// Stop stops the registry background refresh.
func (r *Registry) Stop() {
	close(r.stopRefresh)
}

// Refresh reloads all data from the database.
func (r *Registry) Refresh(ctx context.Context) error {
	// Load providers with models
	providers, err := r.repo.ListProvidersWithModels(ctx, true)
	if err != nil {
		return fmt.Errorf("load providers: %w", err)
	}

	// Build indexes
	providerMap := make(map[uuid.UUID]*Provider)
	providersByType := make(map[ProviderType][]*Provider)
	modelMap := make(map[string]*Model)
	modelsByCapability := make(map[Capability][]*Model)
	modelsByProvider := make(map[uuid.UUID][]*Model)

	for _, p := range providers {
		providerMap[p.ID] = p
		providersByType[p.Type] = append(providersByType[p.Type], p)

		for _, m := range p.Models {
			if !m.Enabled {
				continue
			}
			modelMap[m.ID] = m
			modelsByProvider[p.ID] = append(modelsByProvider[p.ID], m)

			for _, cap := range m.Capabilities {
				capability := Capability(cap)
				modelsByCapability[capability] = append(modelsByCapability[capability], m)
			}
		}
	}

	// Update atomically
	r.mu.Lock()
	r.providers = providerMap
	r.providersByType = providersByType
	r.models = modelMap
	r.modelsByCapability = modelsByCapability
	r.modelsByProvider = modelsByProvider
	r.mu.Unlock()

	return nil
}

// refreshLoop periodically refreshes the registry.
func (r *Registry) refreshLoop() {
	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopRefresh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = r.Refresh(ctx)
			cancel()
		}
	}
}

// GetProvider returns a provider by ID.
func (r *Registry) GetProvider(id uuid.UUID) (*Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

// GetProvidersByType returns providers by type.
func (r *Registry) GetProvidersByType(providerType ProviderType) []*Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providersByType[providerType]
}

// GetModel returns a model by ID.
func (r *Registry) GetModel(id string) (*Model, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[id]
	return m, ok
}

// GetModelWithProvider returns a model with its provider.
func (r *Registry) GetModelWithProvider(id string) (*Model, *Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.models[id]
	if !ok {
		return nil, nil, false
	}

	p, ok := r.providers[m.ProviderID]
	if !ok {
		return nil, nil, false
	}

	return m, p, true
}

// GetModelsByCapability returns models with a specific capability.
func (r *Registry) GetModelsByCapability(cap Capability) []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modelsByCapability[cap]
}

// GetModelsByCapabilities returns models with all specified capabilities.
func (r *Registry) GetModelsByCapabilities(caps []Capability) []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(caps) == 0 {
		// Return all models
		models := make([]*Model, 0, len(r.models))
		for _, m := range r.models {
			models = append(models, m)
		}
		return models
	}

	// Get models with first capability
	candidates := r.modelsByCapability[caps[0]]
	if len(candidates) == 0 {
		return nil
	}

	// Filter by remaining capabilities
	var result []*Model
	for _, m := range candidates {
		if m.HasAllCapabilities(caps) {
			result = append(result, m)
		}
	}

	return result
}

// GetModelsByProvider returns models for a provider.
func (r *Registry) GetModelsByProvider(providerID uuid.UUID) []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modelsByProvider[providerID]
}

// AllProviders returns all providers.
func (r *Registry) AllProviders() []*Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]*Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// AllModels returns all models.
func (r *Registry) AllModels() []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]*Model, 0, len(r.models))
	for _, m := range r.models {
		models = append(models, m)
	}
	return models
}
