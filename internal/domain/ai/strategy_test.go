package ai

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/uniedit/server/internal/model"
)

// ===== Test Helpers =====

func newTestProvider(id uuid.UUID, name string, enabled bool) *model.AIProvider {
	return &model.AIProvider{
		ID:       id,
		Name:     name,
		Type:     model.AIProviderTypeOpenAI,
		BaseURL:  "https://api.openai.com/v1",
		Enabled:  enabled,
		Weight:   1,
		Priority: 0,
	}
}

func newTestModel(id string, providerID uuid.UUID, caps []model.AICapability, contextWindow int) *model.AIModel {
	capStrings := make(pq.StringArray, len(caps))
	for i, c := range caps {
		capStrings[i] = string(c)
	}
	return &model.AIModel{
		ID:              id,
		ProviderID:      providerID,
		Name:            id,
		Capabilities:    capStrings,
		ContextWindow:   contextWindow,
		MaxOutputTokens: 4096,
		InputCostPer1K:  0.01,
		OutputCostPer1K: 0.03,
		Enabled:         true,
	}
}

func newTestCandidate(provider *model.AIProvider, m *model.AIModel) *model.AIScoredCandidate {
	return model.NewAIScoredCandidate(provider, m)
}

// ===== BaseStrategy Tests =====

func TestBaseStrategy(t *testing.T) {
	t.Run("name and priority", func(t *testing.T) {
		bs := NewBaseStrategy("test", 50)
		assert.Equal(t, "test", bs.Name())
		assert.Equal(t, 50, bs.Priority())
	})

	t.Run("filter passes through", func(t *testing.T) {
		bs := NewBaseStrategy("test", 50)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(nil, nil),
			newTestCandidate(nil, nil),
		}
		result := bs.Filter(model.NewAIRoutingContext(), candidates)
		assert.Len(t, result, 2)
	})

	t.Run("score passes through", func(t *testing.T) {
		bs := NewBaseStrategy("test", 50)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(nil, nil),
		}
		result := bs.Score(model.NewAIRoutingContext(), candidates)
		assert.Len(t, result, 1)
	})
}

// ===== UserPreferenceStrategy Tests =====

func TestUserPreferenceStrategy(t *testing.T) {
	strategy := NewUserPreferenceStrategy()

	t.Run("name and priority", func(t *testing.T) {
		assert.Equal(t, "user_preference", strategy.Name())
		assert.Equal(t, 100, strategy.Priority())
	})

	t.Run("filter no preferences returns all", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
		}

		ctx := model.NewAIRoutingContext()
		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 2)
	})

	t.Run("filter with preferences", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
			newTestCandidate(provider, newTestModel("gpt-4o", providerID, []model.AICapability{model.AICapabilityChat}, 128000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.PreferredModels = []string{"gpt-4", "gpt-4o"}
		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 2)
		assert.Equal(t, "gpt-4", result[0].Model.ID)
		assert.Equal(t, "gpt-4o", result[1].Model.ID)
	})

	t.Run("filter fallback when no match", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.PreferredModels = []string{"claude-3"}
		result := strategy.Filter(ctx, candidates)

		// Should return all candidates when no preference matches
		assert.Len(t, result, 2)
	})

	t.Run("score with preferences", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.PreferredModels = []string{"gpt-4", "gpt-3.5"}
		result := strategy.Score(ctx, candidates)

		// First preference should have higher score
		assert.Greater(t, result[0].Score, result[1].Score)
	})
}

// ===== HealthFilterStrategy Tests =====

func TestHealthFilterStrategy(t *testing.T) {
	strategy := NewHealthFilterStrategy()

	t.Run("name and priority", func(t *testing.T) {
		assert.Equal(t, "health_filter", strategy.Name())
		assert.Equal(t, 90, strategy.Priority())
	})

	t.Run("filter no health info returns all", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
		}

		ctx := model.NewAIRoutingContext()
		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 1)
	})

	t.Run("filter removes unhealthy", func(t *testing.T) {
		providerID1 := uuid.New()
		providerID2 := uuid.New()
		provider1 := newTestProvider(providerID1, "openai", true)
		provider2 := newTestProvider(providerID2, "anthropic", true)

		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider1, newTestModel("gpt-4", providerID1, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider2, newTestModel("claude-3", providerID2, []model.AICapability{model.AICapabilityChat}, 200000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.ProviderHealth[providerID1.String()] = true
		ctx.ProviderHealth[providerID2.String()] = false

		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 1)
		assert.Equal(t, "gpt-4", result[0].Model.ID)
	})

	t.Run("filter fallback when all unhealthy", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.ProviderHealth[providerID.String()] = false

		result := strategy.Filter(ctx, candidates)

		// Should return all candidates as fallback
		assert.Len(t, result, 1)
	})

	t.Run("filter unknown health assumes healthy", func(t *testing.T) {
		providerID1 := uuid.New()
		providerID2 := uuid.New()
		provider1 := newTestProvider(providerID1, "openai", true)
		provider2 := newTestProvider(providerID2, "anthropic", true)

		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider1, newTestModel("gpt-4", providerID1, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider2, newTestModel("claude-3", providerID2, []model.AICapability{model.AICapabilityChat}, 200000)),
		}

		ctx := model.NewAIRoutingContext()
		// Only provider1 has health info
		ctx.ProviderHealth[providerID1.String()] = true
		// providerID2 has no health info - should assume healthy

		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 2)
	})
}

// ===== CapabilityFilterStrategy Tests =====

func TestCapabilityFilterStrategy(t *testing.T) {
	strategy := NewCapabilityFilterStrategy()

	t.Run("name and priority", func(t *testing.T) {
		assert.Equal(t, "capability_filter", strategy.Name())
		assert.Equal(t, 80, strategy.Priority())
	})

	t.Run("filter no requirements returns all", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.TaskType = ""
		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 1)
	})

	t.Run("filter by vision capability", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-4-vision", providerID, []model.AICapability{model.AICapabilityChat, model.AICapabilityVision}, 8000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.RequireVision = true

		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 1)
		assert.Equal(t, "gpt-4-vision", result[0].Model.ID)
	})

	t.Run("filter by tools capability", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
			newTestCandidate(provider, newTestModel("gpt-4-turbo", providerID, []model.AICapability{model.AICapabilityChat, model.AICapabilityTools}, 128000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.RequireTools = true

		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 1)
		assert.Equal(t, "gpt-4-turbo", result[0].Model.ID)
	})

	t.Run("filter by multiple capabilities", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-4-vision", providerID, []model.AICapability{model.AICapabilityChat, model.AICapabilityVision}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-4o", providerID, []model.AICapability{model.AICapabilityChat, model.AICapabilityVision, model.AICapabilityTools}, 128000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.RequireVision = true
		ctx.RequireTools = true

		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 1)
		assert.Equal(t, "gpt-4o", result[0].Model.ID)
	})
}

// ===== ContextWindowStrategy Tests =====

func TestContextWindowStrategy(t *testing.T) {
	strategy := NewContextWindowStrategy()

	t.Run("name and priority", func(t *testing.T) {
		assert.Equal(t, "context_window", strategy.Name())
		assert.Equal(t, 70, strategy.Priority())
	})

	t.Run("filter no requirements returns all", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-4o", providerID, []model.AICapability{model.AICapabilityChat}, 128000)),
		}

		ctx := model.NewAIRoutingContext()
		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 2)
	})

	t.Run("filter by estimated tokens", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-4o", providerID, []model.AICapability{model.AICapabilityChat}, 128000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.EstimatedTokens = 10000 // Needs 10000 + 4096 buffer = 14096

		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 1)
		assert.Equal(t, "gpt-4o", result[0].Model.ID)
	})

	t.Run("filter by min context window", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-4o", providerID, []model.AICapability{model.AICapabilityChat}, 128000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.MinContextWindow = 16000

		result := strategy.Filter(ctx, candidates)

		assert.Len(t, result, 1)
		assert.Equal(t, "gpt-4o", result[0].Model.ID)
	})

	t.Run("filter fallback to largest context", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.MinContextWindow = 200000 // No model can satisfy

		result := strategy.Filter(ctx, candidates)

		// Should return models with largest context
		assert.Len(t, result, 1)
		assert.Equal(t, "gpt-4", result[0].Model.ID)
	})

	t.Run("score by context window", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
			newTestCandidate(provider, newTestModel("gpt-4o", providerID, []model.AICapability{model.AICapabilityChat}, 128000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.EstimatedTokens = 1000

		result := strategy.Score(ctx, candidates)

		// Larger context should have higher score
		assert.Greater(t, result[1].Score, result[0].Score)
	})
}

// ===== CostOptimizationStrategy Tests =====

func TestCostOptimizationStrategy(t *testing.T) {
	strategy := NewCostOptimizationStrategy()

	t.Run("name and priority", func(t *testing.T) {
		assert.Equal(t, "cost_optimization", strategy.Name())
		assert.Equal(t, 50, strategy.Priority())
	})

	t.Run("score not cost optimized returns unchanged", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)

		m1 := newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)
		m1.InputCostPer1K = 0.03
		m1.OutputCostPer1K = 0.06

		m2 := newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)
		m2.InputCostPer1K = 0.001
		m2.OutputCostPer1K = 0.002

		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, m1),
			newTestCandidate(provider, m2),
		}

		ctx := model.NewAIRoutingContext()
		ctx.Optimize = "quality" // Not cost

		result := strategy.Score(ctx, candidates)

		// Scores should not change
		assert.Equal(t, float64(0), result[0].Score)
		assert.Equal(t, float64(0), result[1].Score)
	})

	t.Run("score cost optimized prefers cheaper", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)

		m1 := newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)
		m1.InputCostPer1K = 0.03
		m1.OutputCostPer1K = 0.06

		m2 := newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)
		m2.InputCostPer1K = 0.001
		m2.OutputCostPer1K = 0.002

		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, m1),
			newTestCandidate(provider, m2),
		}

		ctx := model.NewAIRoutingContext()
		ctx.Optimize = "cost"

		result := strategy.Score(ctx, candidates)

		// Cheaper model (gpt-3.5) should have higher score
		assert.Greater(t, result[1].Score, result[0].Score)
	})
}

// ===== LoadBalancingStrategy Tests =====

func TestLoadBalancingStrategy(t *testing.T) {
	strategy := NewLoadBalancingStrategy()

	t.Run("name and priority", func(t *testing.T) {
		assert.Equal(t, "load_balancing", strategy.Name())
		assert.Equal(t, 10, strategy.Priority())
	})

	t.Run("score adds jitter", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
		}

		ctx := model.NewAIRoutingContext()
		result := strategy.Score(ctx, candidates)

		// All candidates should have small positive scores (jitter)
		for _, c := range result {
			assert.Greater(t, c.Score, float64(0))
			assert.Less(t, c.Score, float64(0.1))
		}
	})
}

// ===== StrategyChain Tests =====

func TestStrategyChain(t *testing.T) {
	t.Run("strategies sorted by priority", func(t *testing.T) {
		chain := NewStrategyChain(
			NewLoadBalancingStrategy(),   // Priority 10
			NewHealthFilterStrategy(),    // Priority 90
			NewUserPreferenceStrategy(),  // Priority 100
		)

		// Verify internal order (should be sorted descending by priority)
		assert.Equal(t, "user_preference", chain.strategies[0].Name())
		assert.Equal(t, "health_filter", chain.strategies[1].Name())
		assert.Equal(t, "load_balancing", chain.strategies[2].Name())
	})

	t.Run("execute empty candidates error", func(t *testing.T) {
		chain := DefaultStrategyChain()

		ctx := model.NewAIRoutingContext()
		result, err := chain.Execute(ctx, []*model.AIScoredCandidate{})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no candidates")
	})

	t.Run("execute returns best candidate", func(t *testing.T) {
		chain := NewStrategyChain(
			NewUserPreferenceStrategy(),
		)

		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)

		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
			newTestCandidate(provider, newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.PreferredModels = []string{"gpt-4"}

		result, err := chain.Execute(ctx, candidates)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "gpt-4", result.Model.ID)
	})

	t.Run("add strategy maintains order", func(t *testing.T) {
		chain := NewStrategyChain(
			NewUserPreferenceStrategy(), // Priority 100
		)

		chain.AddStrategy(NewHealthFilterStrategy())     // Priority 90
		chain.AddStrategy(NewLoadBalancingStrategy())    // Priority 10

		// Verify order is maintained
		assert.Equal(t, "user_preference", chain.strategies[0].Name())
		assert.Equal(t, "health_filter", chain.strategies[1].Name())
		assert.Equal(t, "load_balancing", chain.strategies[2].Name())
	})

	t.Run("execute filter eliminates all returns error", func(t *testing.T) {
		chain := NewStrategyChain(
			NewCapabilityFilterStrategy(),
		)

		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)

		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider, newTestModel("gpt-3.5", providerID, []model.AICapability{model.AICapabilityChat}, 4000)),
		}

		ctx := model.NewAIRoutingContext()
		ctx.RequireVision = true // No model has vision

		result, err := chain.Execute(ctx, candidates)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no candidates after")
	})
}

// ===== DefaultStrategyChain Tests =====

func TestDefaultStrategyChain(t *testing.T) {
	t.Run("has all strategies", func(t *testing.T) {
		chain := DefaultStrategyChain()

		assert.Len(t, chain.strategies, 6)

		names := make([]string, len(chain.strategies))
		for i, s := range chain.strategies {
			names[i] = s.Name()
		}

		assert.Contains(t, names, "user_preference")
		assert.Contains(t, names, "health_filter")
		assert.Contains(t, names, "capability_filter")
		assert.Contains(t, names, "context_window")
		assert.Contains(t, names, "cost_optimization")
		assert.Contains(t, names, "load_balancing")
	})

	t.Run("integrates all strategies", func(t *testing.T) {
		chain := DefaultStrategyChain()

		providerID1 := uuid.New()
		providerID2 := uuid.New()
		provider1 := newTestProvider(providerID1, "openai", true)
		provider2 := newTestProvider(providerID2, "anthropic", true)

		m1 := newTestModel("gpt-4", providerID1, []model.AICapability{model.AICapabilityChat, model.AICapabilityVision}, 128000)
		m1.InputCostPer1K = 0.03
		m1.OutputCostPer1K = 0.06

		m2 := newTestModel("gpt-3.5", providerID1, []model.AICapability{model.AICapabilityChat}, 16000)
		m2.InputCostPer1K = 0.001
		m2.OutputCostPer1K = 0.002

		m3 := newTestModel("claude-3", providerID2, []model.AICapability{model.AICapabilityChat, model.AICapabilityVision}, 200000)
		m3.InputCostPer1K = 0.01
		m3.OutputCostPer1K = 0.03

		candidates := []*model.AIScoredCandidate{
			newTestCandidate(provider1, m1),
			newTestCandidate(provider1, m2),
			newTestCandidate(provider2, m3),
		}

		ctx := model.NewAIRoutingContext()
		ctx.RequireVision = true
		ctx.ProviderHealth[providerID1.String()] = true
		ctx.ProviderHealth[providerID2.String()] = true

		result, err := chain.Execute(ctx, candidates)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Should select a model with vision capability
		assert.True(t, result.Model.HasCapability(model.AICapabilityVision))
	})
}

// ===== AIRoutingContext Tests =====

func TestAIRoutingContext(t *testing.T) {
	t.Run("new context defaults", func(t *testing.T) {
		ctx := model.NewAIRoutingContext()

		assert.Equal(t, "chat", ctx.TaskType)
		assert.NotNil(t, ctx.ProviderHealth)
		assert.NotNil(t, ctx.Metadata)
	})

	t.Run("required capabilities chat only", func(t *testing.T) {
		ctx := model.NewAIRoutingContext()

		caps := ctx.RequiredCapabilities()

		assert.Len(t, caps, 1)
		assert.Contains(t, caps, model.AICapabilityChat)
	})

	t.Run("required capabilities with stream", func(t *testing.T) {
		ctx := model.NewAIRoutingContext()
		ctx.RequireStream = true

		caps := ctx.RequiredCapabilities()

		assert.Len(t, caps, 2)
		assert.Contains(t, caps, model.AICapabilityChat)
		assert.Contains(t, caps, model.AICapabilityStream)
	})

	t.Run("required capabilities all", func(t *testing.T) {
		ctx := model.NewAIRoutingContext()
		ctx.RequireStream = true
		ctx.RequireTools = true
		ctx.RequireVision = true
		ctx.RequireJSON = true

		caps := ctx.RequiredCapabilities()

		assert.Len(t, caps, 5)
		assert.Contains(t, caps, model.AICapabilityChat)
		assert.Contains(t, caps, model.AICapabilityStream)
		assert.Contains(t, caps, model.AICapabilityTools)
		assert.Contains(t, caps, model.AICapabilityVision)
		assert.Contains(t, caps, model.AICapabilityJSON)
	})
}

// ===== AIScoredCandidate Tests =====

func TestAIScoredCandidate(t *testing.T) {
	t.Run("new candidate", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		m := newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)

		candidate := model.NewAIScoredCandidate(provider, m)

		assert.Equal(t, provider, candidate.Provider)
		assert.Equal(t, m, candidate.Model)
		assert.Equal(t, float64(0), candidate.Score)
		assert.NotNil(t, candidate.ScoreBreakdown)
		assert.NotNil(t, candidate.Reasons)
	})

	t.Run("add score", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		m := newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)

		candidate := model.NewAIScoredCandidate(provider, m)
		candidate.AddScore("test_strategy", 10.5, "test reason")

		assert.Equal(t, 10.5, candidate.Score)
		assert.Equal(t, 10.5, candidate.ScoreBreakdown["test_strategy"])
		assert.Contains(t, candidate.Reasons, "test reason")
	})

	t.Run("add multiple scores", func(t *testing.T) {
		providerID := uuid.New()
		provider := newTestProvider(providerID, "openai", true)
		m := newTestModel("gpt-4", providerID, []model.AICapability{model.AICapabilityChat}, 8000)

		candidate := model.NewAIScoredCandidate(provider, m)
		candidate.AddScore("strategy1", 5.0, "reason1")
		candidate.AddScore("strategy2", 3.0, "reason2")

		assert.Equal(t, 8.0, candidate.Score)
		assert.Equal(t, 5.0, candidate.ScoreBreakdown["strategy1"])
		assert.Equal(t, 3.0, candidate.ScoreBreakdown["strategy2"])
		assert.Len(t, candidate.Reasons, 2)
	})
}
