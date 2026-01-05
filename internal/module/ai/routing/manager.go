package routing

import (
	"context"
	"fmt"

	"github.com/uniedit/server/internal/module/ai/group"
	"github.com/uniedit/server/internal/module/ai/provider"
)

// Manager handles routing decisions.
type Manager struct {
	registry      *provider.Registry
	healthMonitor *provider.HealthMonitor
	groupManager  *group.Manager
	chain         *Chain
}

// ManagerConfig contains manager configuration.
type ManagerConfig struct {
	// Custom chain to use (if nil, DefaultChain is used)
	Chain *Chain
}

// NewManager creates a new routing manager.
func NewManager(
	registry *provider.Registry,
	healthMonitor *provider.HealthMonitor,
	groupManager *group.Manager,
	config *ManagerConfig,
) *Manager {
	chain := DefaultChain()
	if config != nil && config.Chain != nil {
		chain = config.Chain
	}

	return &Manager{
		registry:      registry,
		healthMonitor: healthMonitor,
		groupManager:  groupManager,
		chain:         chain,
	}
}

// Route selects the best model for the given context.
func (m *Manager) Route(ctx context.Context, routingCtx *Context) (*Result, error) {
	// Get candidates
	candidates, err := m.getCandidates(ctx, routingCtx)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available models for routing")
	}

	// Inject health status
	if m.healthMonitor != nil {
		healthStatus := m.healthMonitor.AllHealthStatus()
		for id, status := range healthStatus {
			routingCtx.ProviderHealth[id.String()] = status == provider.HealthStatusHealthy
		}
	}

	// Execute strategy chain
	return m.chain.Execute(routingCtx, candidates)
}

// RouteWithFallback routes with fallback support.
func (m *Manager) RouteWithFallback(ctx context.Context, routingCtx *Context, execute func(*Result) error) (*Result, error) {
	// Get group if specified
	var grp *group.Group
	var err error
	if routingCtx.GroupID != "" {
		grp, err = m.groupManager.Get(ctx, routingCtx.GroupID)
		if err != nil {
			return nil, fmt.Errorf("get group: %w", err)
		}
	}

	// Try routing
	result, err := m.Route(ctx, routingCtx)
	if err != nil {
		return nil, err
	}

	// Execute
	if err := execute(result); err != nil {
		// Check if fallback is enabled
		if grp != nil && grp.Fallback != nil && grp.Fallback.Enabled {
			return m.executeFallback(ctx, routingCtx, grp, result, execute, err)
		}
		return nil, err
	}

	return result, nil
}

// executeFallback tries fallback models.
func (m *Manager) executeFallback(
	ctx context.Context,
	routingCtx *Context,
	grp *group.Group,
	failedResult *Result,
	execute func(*Result) error,
	lastErr error,
) (*Result, error) {
	maxAttempts := grp.Fallback.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	// Track failed models
	failedModels := map[string]bool{
		failedResult.Model.ID: true,
	}

	for attempt := 1; attempt < maxAttempts; attempt++ {
		// Exclude failed models
		var excludeModels []string
		for modelID := range failedModels {
			excludeModels = append(excludeModels, modelID)
		}

		// Create new context with exclusions
		fallbackCtx := *routingCtx
		fallbackCtx.PreferredModels = nil

		// Route again
		result, err := m.Route(ctx, &fallbackCtx)
		if err != nil {
			continue
		}

		// Skip if already failed
		if failedModels[result.Model.ID] {
			continue
		}

		// Try execute
		if err := execute(result); err != nil {
			failedModels[result.Model.ID] = true
			lastErr = err
			continue
		}

		return result, nil
	}

	return nil, fmt.Errorf("all fallback attempts failed: %w", lastErr)
}

// getCandidates builds the list of candidate models.
func (m *Manager) getCandidates(ctx context.Context, routingCtx *Context) ([]*ScoredCandidate, error) {
	var models []*provider.Model

	// If group is specified, use group models
	if routingCtx.GroupID != "" {
		grp, err := m.groupManager.Get(ctx, routingCtx.GroupID)
		if err != nil {
			return nil, fmt.Errorf("get group: %w", err)
		}

		for _, modelID := range grp.Models {
			if model, ok := m.registry.GetModel(modelID); ok {
				models = append(models, model)
			}
		}
	} else {
		// Get all models with required capabilities
		caps := routingCtx.RequiredCapabilities()
		models = m.registry.GetModelsByCapabilities(caps)
	}

	// Build scored candidates
	candidates := make([]*ScoredCandidate, 0, len(models))
	for _, model := range models {
		prov, ok := m.registry.GetProvider(model.ProviderID)
		if !ok || !prov.Enabled {
			continue
		}

		candidates = append(candidates, NewScoredCandidate(prov, model))
	}

	return candidates, nil
}

// SetChain sets a custom strategy chain.
func (m *Manager) SetChain(chain *Chain) {
	m.chain = chain
}
