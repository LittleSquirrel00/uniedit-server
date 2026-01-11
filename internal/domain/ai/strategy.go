package ai

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/uniedit/server/internal/model"
)

// Strategy defines the interface for routing strategies.
type Strategy interface {
	// Name returns the strategy name.
	Name() string

	// Priority returns the priority (higher = runs first).
	Priority() int

	// Filter filters candidates.
	Filter(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate

	// Score scores candidates.
	Score(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate
}

// BaseStrategy provides common functionality for strategies.
type BaseStrategy struct {
	name     string
	priority int
}

// NewBaseStrategy creates a new base strategy.
func NewBaseStrategy(name string, priority int) *BaseStrategy {
	return &BaseStrategy{
		name:     name,
		priority: priority,
	}
}

// Name returns the strategy name.
func (s *BaseStrategy) Name() string {
	return s.name
}

// Priority returns the strategy priority.
func (s *BaseStrategy) Priority() int {
	return s.priority
}

// Filter is a no-op filter (override in implementations).
func (s *BaseStrategy) Filter(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	return candidates
}

// Score is a no-op scorer (override in implementations).
func (s *BaseStrategy) Score(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	return candidates
}

// StrategyChain executes strategies in priority order.
type StrategyChain struct {
	strategies []Strategy
}

// NewStrategyChain creates a new strategy chain.
func NewStrategyChain(strategies ...Strategy) *StrategyChain {
	// Sort by priority (descending)
	sorted := make([]Strategy, len(strategies))
	copy(sorted, strategies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() > sorted[j].Priority()
	})

	return &StrategyChain{strategies: sorted}
}

// Execute runs the strategy chain and returns the best candidate.
func (c *StrategyChain) Execute(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) (*model.AIRoutingResult, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates provided")
	}

	result := candidates

	// Execute each strategy in order
	for _, strategy := range c.strategies {
		// Filter
		result = strategy.Filter(ctx, result)
		if len(result) == 0 {
			return nil, fmt.Errorf("no candidates after %s filter", strategy.Name())
		}

		// Score
		result = strategy.Score(ctx, result)
	}

	// Sort by score (descending)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	best := result[0]
	return &model.AIRoutingResult{
		Provider: best.Provider,
		Model:    best.Model,
		Score:    best.Score,
		Reason:   strings.Join(best.Reasons, "; "),
	}, nil
}

// AddStrategy adds a strategy to the chain.
func (c *StrategyChain) AddStrategy(strategy Strategy) {
	c.strategies = append(c.strategies, strategy)
	// Re-sort
	sort.Slice(c.strategies, func(i, j int) bool {
		return c.strategies[i].Priority() > c.strategies[j].Priority()
	})
}

// DefaultStrategyChain creates a chain with all default strategies.
func DefaultStrategyChain() *StrategyChain {
	return NewStrategyChain(
		NewUserPreferenceStrategy(),
		NewHealthFilterStrategy(),
		NewCapabilityFilterStrategy(),
		NewContextWindowStrategy(),
		NewCostOptimizationStrategy(),
		NewLoadBalancingStrategy(),
	)
}

// ==================== User Preference Strategy ====================

const (
	userPreferenceName     = "user_preference"
	userPreferencePriority = 100
)

// UserPreferenceStrategy prioritizes user-specified models.
type UserPreferenceStrategy struct {
	*BaseStrategy
}

// NewUserPreferenceStrategy creates a new user preference strategy.
func NewUserPreferenceStrategy() *UserPreferenceStrategy {
	return &UserPreferenceStrategy{
		BaseStrategy: NewBaseStrategy(userPreferenceName, userPreferencePriority),
	}
}

// Filter filters out non-preferred models if preferences exist.
func (s *UserPreferenceStrategy) Filter(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	if len(ctx.PreferredModels) == 0 {
		return candidates
	}

	preferredSet := make(map[string]bool)
	for _, m := range ctx.PreferredModels {
		preferredSet[m] = true
	}

	var result []*model.AIScoredCandidate
	for _, c := range candidates {
		if preferredSet[c.Model.ID] {
			result = append(result, c)
		}
	}

	// If no preferred models found, return all candidates
	if len(result) == 0 {
		return candidates
	}

	return result
}

// Score gives higher scores to preferred models.
func (s *UserPreferenceStrategy) Score(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	if len(ctx.PreferredModels) == 0 {
		return candidates
	}

	preferenceOrder := make(map[string]int)
	for i, m := range ctx.PreferredModels {
		preferenceOrder[m] = len(ctx.PreferredModels) - i
	}

	for _, c := range candidates {
		if order, ok := preferenceOrder[c.Model.ID]; ok {
			bonus := float64(order) * 10
			c.AddScore(userPreferenceName, bonus, "user preference")
		}
	}

	return candidates
}

// ==================== Health Filter Strategy ====================

const (
	healthFilterName     = "health_filter"
	healthFilterPriority = 90
)

// HealthFilterStrategy filters out unhealthy providers.
type HealthFilterStrategy struct {
	*BaseStrategy
}

// NewHealthFilterStrategy creates a new health filter strategy.
func NewHealthFilterStrategy() *HealthFilterStrategy {
	return &HealthFilterStrategy{
		BaseStrategy: NewBaseStrategy(healthFilterName, healthFilterPriority),
	}
}

// Filter removes candidates with unhealthy providers.
func (s *HealthFilterStrategy) Filter(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	if len(ctx.ProviderHealth) == 0 {
		return candidates
	}

	var result []*model.AIScoredCandidate
	for _, c := range candidates {
		providerID := c.Provider.ID.String()
		if healthy, ok := ctx.ProviderHealth[providerID]; ok && healthy {
			result = append(result, c)
		} else if !ok {
			// Unknown health status, assume healthy
			result = append(result, c)
		}
	}

	// If all providers are unhealthy, return all candidates as fallback
	if len(result) == 0 {
		return candidates
	}

	return result
}

// ==================== Capability Filter Strategy ====================

const (
	capabilityFilterName     = "capability_filter"
	capabilityFilterPriority = 80
)

// CapabilityFilterStrategy filters models that don't meet capability requirements.
type CapabilityFilterStrategy struct {
	*BaseStrategy
}

// NewCapabilityFilterStrategy creates a new capability filter strategy.
func NewCapabilityFilterStrategy() *CapabilityFilterStrategy {
	return &CapabilityFilterStrategy{
		BaseStrategy: NewBaseStrategy(capabilityFilterName, capabilityFilterPriority),
	}
}

// Filter removes candidates that don't have required capabilities.
func (s *CapabilityFilterStrategy) Filter(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	requiredCaps := ctx.RequiredCapabilities()
	if len(requiredCaps) == 0 {
		return candidates
	}

	var result []*model.AIScoredCandidate
	for _, c := range candidates {
		if c.Model.HasAllCapabilities(requiredCaps) {
			result = append(result, c)
		}
	}

	return result
}

// ==================== Context Window Strategy ====================

const (
	contextWindowName     = "context_window"
	contextWindowPriority = 70
)

// ContextWindowStrategy filters models based on context window size.
type ContextWindowStrategy struct {
	*BaseStrategy
}

// NewContextWindowStrategy creates a new context window strategy.
func NewContextWindowStrategy() *ContextWindowStrategy {
	return &ContextWindowStrategy{
		BaseStrategy: NewBaseStrategy(contextWindowName, contextWindowPriority),
	}
}

// Filter removes models that can't handle the input size.
func (s *ContextWindowStrategy) Filter(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	if ctx.EstimatedTokens == 0 && ctx.MinContextWindow == 0 {
		return candidates
	}

	// Add buffer for response
	requiredContext := ctx.EstimatedTokens + 4096 // Default buffer for response
	if ctx.MinContextWindow > requiredContext {
		requiredContext = ctx.MinContextWindow
	}

	var result []*model.AIScoredCandidate
	for _, c := range candidates {
		if c.Model.ContextWindow >= requiredContext {
			result = append(result, c)
		}
	}

	// If no models can handle it, return models with largest context
	if len(result) == 0 {
		maxContext := 0
		for _, c := range candidates {
			if c.Model.ContextWindow > maxContext {
				maxContext = c.Model.ContextWindow
			}
		}
		for _, c := range candidates {
			if c.Model.ContextWindow == maxContext {
				result = append(result, c)
			}
		}
	}

	return result
}

// Score gives higher scores to models with larger context windows.
func (s *ContextWindowStrategy) Score(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	if ctx.EstimatedTokens == 0 && ctx.MinContextWindow == 0 {
		return candidates
	}

	// Find max context window
	maxContext := 0
	for _, c := range candidates {
		if c.Model.ContextWindow > maxContext {
			maxContext = c.Model.ContextWindow
		}
	}

	if maxContext == 0 {
		return candidates
	}

	// Score based on relative context window size
	for _, c := range candidates {
		// Small bonus for having adequate context
		ratio := float64(c.Model.ContextWindow) / float64(maxContext)
		bonus := ratio * 5
		c.AddScore(contextWindowName, bonus, "context window")
	}

	return candidates
}

// ==================== Cost Optimization Strategy ====================

const (
	costOptimizationName     = "cost_optimization"
	costOptimizationPriority = 50
)

// CostOptimizationStrategy scores models based on cost.
type CostOptimizationStrategy struct {
	*BaseStrategy
}

// NewCostOptimizationStrategy creates a new cost optimization strategy.
func NewCostOptimizationStrategy() *CostOptimizationStrategy {
	return &CostOptimizationStrategy{
		BaseStrategy: NewBaseStrategy(costOptimizationName, costOptimizationPriority),
	}
}

// Score gives higher scores to cheaper models.
func (s *CostOptimizationStrategy) Score(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	// Skip if not optimizing for cost
	if ctx.Optimize != "cost" {
		return candidates
	}

	// Find min cost
	minCost := float64(-1)
	for _, c := range candidates {
		cost := c.Model.InputCostPer1K + c.Model.OutputCostPer1K
		if minCost < 0 || cost < minCost {
			minCost = cost
		}
	}

	if minCost <= 0 {
		return candidates
	}

	// Score based on relative cost (lower cost = higher score)
	for _, c := range candidates {
		cost := c.Model.InputCostPer1K + c.Model.OutputCostPer1K
		if cost > 0 {
			ratio := minCost / cost
			bonus := ratio * 20
			c.AddScore(costOptimizationName, bonus, "cost optimization")
		}
	}

	return candidates
}

// ==================== Load Balancing Strategy ====================

const (
	loadBalancingName     = "load_balancing"
	loadBalancingPriority = 10
)

// LoadBalancingStrategy adds random jitter to distribute load.
type LoadBalancingStrategy struct {
	*BaseStrategy
}

// NewLoadBalancingStrategy creates a new load balancing strategy.
func NewLoadBalancingStrategy() *LoadBalancingStrategy {
	return &LoadBalancingStrategy{
		BaseStrategy: NewBaseStrategy(loadBalancingName, loadBalancingPriority),
	}
}

// Score adds small random jitter for load distribution.
func (s *LoadBalancingStrategy) Score(ctx *model.AIRoutingContext, candidates []*model.AIScoredCandidate) []*model.AIScoredCandidate {
	for _, c := range candidates {
		jitter := rand.Float64() * 0.1
		c.AddScore(loadBalancingName, jitter, "load balancing")
	}
	return candidates
}
