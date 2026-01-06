package routing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/ai/group"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/ai/provider/pool"
)

// Manager handles routing decisions.
type Manager struct {
	registry      *provider.Registry
	healthMonitor *provider.HealthMonitor
	groupManager  *group.Manager
	chain         *Chain
	accountPool   *pool.Manager // Optional account pool manager
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

// SetAccountPool sets the account pool manager for multi-account support.
func (m *Manager) SetAccountPool(accountPool *pool.Manager) {
	m.accountPool = accountPool
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
	result, err := m.chain.Execute(routingCtx, candidates)
	if err != nil {
		return nil, err
	}

	// Get API key from account pool or provider
	if err := m.resolveAPIKey(ctx, result); err != nil {
		return nil, fmt.Errorf("resolve API key: %w", err)
	}

	return result, nil
}

// resolveAPIKey gets the API key from account pool or falls back to provider.
func (m *Manager) resolveAPIKey(ctx context.Context, result *Result) error {
	// Try to get account from pool if available
	if m.accountPool != nil {
		account, err := m.accountPool.GetAccount(ctx, result.Provider.ID)
		if err == nil && account != nil {
			accountID := account.ID.String()
			result.AccountID = &accountID
			result.APIKey = account.DecryptedKey
			return nil
		}
		// If pool returns error (e.g., no accounts), fall back to provider
	}

	// Fall back to provider's API key
	result.APIKey = result.Provider.APIKey
	return nil
}

// MarkAccountSuccess records a successful request for the account.
func (m *Manager) MarkAccountSuccess(ctx context.Context, result *Result, tokens int, costUSD float64) error {
	if m.accountPool == nil || result.AccountID == nil {
		return nil
	}

	accountID, err := uuid.Parse(*result.AccountID)
	if err != nil {
		return err
	}

	return m.accountPool.MarkSuccess(ctx, accountID, tokens, costUSD)
}

// MarkAccountFailure records a failed request for the account.
func (m *Manager) MarkAccountFailure(ctx context.Context, result *Result, reqErr error) error {
	if m.accountPool == nil || result.AccountID == nil {
		return nil
	}

	accountID, err := uuid.Parse(*result.AccountID)
	if err != nil {
		return err
	}

	return m.accountPool.MarkFailure(ctx, accountID, reqErr)
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
