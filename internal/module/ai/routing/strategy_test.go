package routing

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uniedit/server/internal/module/ai/provider"
)

// Test helpers

func newTestProvider(name string) *provider.Provider {
	return &provider.Provider{
		ID:      uuid.New(),
		Name:    name,
		Type:    provider.ProviderTypeOpenAI,
		Enabled: true,
	}
}

func newTestModel(id string, capabilities []string, contextWindow int, inputCost, outputCost float64) *provider.Model {
	return &provider.Model{
		ID:              id,
		Name:            id,
		Capabilities:    capabilities,
		ContextWindow:   contextWindow,
		InputCostPer1K:  inputCost,
		OutputCostPer1K: outputCost,
		Enabled:         true,
	}
}

func newTestCandidates() []*ScoredCandidate {
	p1 := newTestProvider("Provider1")
	p2 := newTestProvider("Provider2")

	m1 := newTestModel("gpt-4o", []string{"chat", "vision"}, 128000, 0.005, 0.015)
	m2 := newTestModel("gpt-4o-mini", []string{"chat", "vision"}, 128000, 0.00015, 0.0006)
	m3 := newTestModel("gpt-3.5-turbo", []string{"chat"}, 16000, 0.0005, 0.0015)
	m4 := newTestModel("claude-3-5-sonnet", []string{"chat", "vision"}, 200000, 0.003, 0.015)

	m1.ProviderID = p1.ID
	m2.ProviderID = p1.ID
	m3.ProviderID = p1.ID
	m4.ProviderID = p2.ID

	return []*ScoredCandidate{
		NewScoredCandidate(p1, m1),
		NewScoredCandidate(p1, m2),
		NewScoredCandidate(p1, m3),
		NewScoredCandidate(p2, m4),
	}
}

// Tests for BaseStrategy

func TestBaseStrategy(t *testing.T) {
	t.Run("Name", func(t *testing.T) {
		s := NewBaseStrategy("test", 50)
		assert.Equal(t, "test", s.Name())
	})

	t.Run("Priority", func(t *testing.T) {
		s := NewBaseStrategy("test", 50)
		assert.Equal(t, 50, s.Priority())
	})

	t.Run("Filter returns all candidates", func(t *testing.T) {
		s := NewBaseStrategy("test", 50)
		candidates := newTestCandidates()
		result := s.Filter(NewContext(), candidates)
		assert.Equal(t, len(candidates), len(result))
	})

	t.Run("Score returns all candidates", func(t *testing.T) {
		s := NewBaseStrategy("test", 50)
		candidates := newTestCandidates()
		result := s.Score(NewContext(), candidates)
		assert.Equal(t, len(candidates), len(result))
	})
}

// Tests for Chain

func TestChain(t *testing.T) {
	t.Run("Execute with empty candidates returns error", func(t *testing.T) {
		chain := NewChain()
		_, err := chain.Execute(NewContext(), []*ScoredCandidate{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no candidates")
	})

	t.Run("Execute returns best candidate", func(t *testing.T) {
		chain := NewChain(NewLoadBalancing())
		candidates := newTestCandidates()
		result, err := chain.Execute(NewContext(), candidates)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Model)
	})

	t.Run("Strategies are sorted by priority", func(t *testing.T) {
		s1 := NewBaseStrategy("low", 10)
		s2 := NewBaseStrategy("high", 100)
		s3 := NewBaseStrategy("medium", 50)

		chain := NewChain(s1, s2, s3)
		assert.Equal(t, "high", chain.strategies[0].Name())
		assert.Equal(t, "medium", chain.strategies[1].Name())
		assert.Equal(t, "low", chain.strategies[2].Name())
	})

	t.Run("AddStrategy maintains order", func(t *testing.T) {
		chain := NewChain(NewBaseStrategy("low", 10))
		chain.AddStrategy(NewBaseStrategy("high", 100))
		assert.Equal(t, "high", chain.strategies[0].Name())
	})
}

// Tests for UserPreference

func TestUserPreference(t *testing.T) {
	strategy := NewUserPreference()

	t.Run("Name and Priority", func(t *testing.T) {
		assert.Equal(t, "user_preference", strategy.Name())
		assert.Equal(t, 100, strategy.Priority())
	})

	t.Run("Filter with no preferences returns all", func(t *testing.T) {
		ctx := NewContext()
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)
		assert.Equal(t, len(candidates), len(result))
	})

	t.Run("Filter with preferences filters candidates", func(t *testing.T) {
		ctx := NewContext()
		ctx.PreferredModels = []string{"gpt-4o", "gpt-4o-mini"}
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)
		assert.Equal(t, 2, len(result))
		for _, c := range result {
			assert.Contains(t, []string{"gpt-4o", "gpt-4o-mini"}, c.Model.ID)
		}
	})

	t.Run("Filter returns all if no preferred models match", func(t *testing.T) {
		ctx := NewContext()
		ctx.PreferredModels = []string{"non-existent"}
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)
		assert.Equal(t, len(candidates), len(result))
	})

	t.Run("Score adds bonus for preferred models", func(t *testing.T) {
		ctx := NewContext()
		ctx.PreferredModels = []string{"gpt-4o"}
		candidates := newTestCandidates()
		result := strategy.Score(ctx, candidates)

		var preferredScore float64
		for _, c := range result {
			if c.Model.ID == "gpt-4o" {
				preferredScore = c.Score
				break
			}
		}
		assert.Greater(t, preferredScore, float64(0))
	})
}

// Tests for HealthFilter

func TestHealthFilter(t *testing.T) {
	strategy := NewHealthFilter()

	t.Run("Name and Priority", func(t *testing.T) {
		assert.Equal(t, "health_filter", strategy.Name())
		assert.Equal(t, 90, strategy.Priority())
	})

	t.Run("Filter with no health info returns all", func(t *testing.T) {
		ctx := NewContext()
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)
		assert.Equal(t, len(candidates), len(result))
	})

	t.Run("Filter removes unhealthy providers", func(t *testing.T) {
		ctx := NewContext()
		candidates := newTestCandidates()

		// Mark first provider as unhealthy
		ctx.ProviderHealth[candidates[0].Provider.ID.String()] = false
		ctx.ProviderHealth[candidates[3].Provider.ID.String()] = true

		result := strategy.Filter(ctx, candidates)
		// Should only have the healthy provider's model
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "claude-3-5-sonnet", result[0].Model.ID)
	})

	t.Run("Filter returns all if all providers unhealthy", func(t *testing.T) {
		ctx := NewContext()
		candidates := newTestCandidates()

		for _, c := range candidates {
			ctx.ProviderHealth[c.Provider.ID.String()] = false
		}

		result := strategy.Filter(ctx, candidates)
		assert.Equal(t, len(candidates), len(result))
	})
}

// Tests for CapabilityFilter

func TestCapabilityFilter(t *testing.T) {
	strategy := NewCapabilityFilter()

	t.Run("Name and Priority", func(t *testing.T) {
		assert.Equal(t, "capability_filter", strategy.Name())
		assert.Equal(t, 80, strategy.Priority())
	})

	t.Run("Filter with no required capabilities returns all", func(t *testing.T) {
		ctx := NewContext()
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)
		assert.Equal(t, len(candidates), len(result))
	})

	t.Run("Filter with vision requirement", func(t *testing.T) {
		ctx := NewContext()
		ctx.RequireVision = true
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)

		// Should only have models with vision capability
		for _, c := range result {
			assert.Contains(t, c.Model.Capabilities, "vision")
		}
	})
}

// Tests for ContextWindow

func TestContextWindow(t *testing.T) {
	strategy := NewContextWindow()

	t.Run("Name and Priority", func(t *testing.T) {
		assert.Equal(t, "context_window", strategy.Name())
		assert.Equal(t, 70, strategy.Priority())
	})

	t.Run("Filter with no requirements returns all", func(t *testing.T) {
		ctx := NewContext()
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)
		assert.Equal(t, len(candidates), len(result))
	})

	t.Run("Filter by estimated tokens", func(t *testing.T) {
		ctx := NewContext()
		ctx.EstimatedTokens = 50000 // Needs at least 54096 with buffer
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)

		// Should exclude gpt-3.5-turbo with 16000 context
		for _, c := range result {
			assert.NotEqual(t, "gpt-3.5-turbo", c.Model.ID)
		}
	})

	t.Run("Filter returns largest context if none fit", func(t *testing.T) {
		ctx := NewContext()
		ctx.EstimatedTokens = 500000 // Too large for any model
		candidates := newTestCandidates()
		result := strategy.Filter(ctx, candidates)

		// Should return model with largest context
		assert.Greater(t, len(result), 0)
		for _, c := range result {
			assert.Equal(t, 200000, c.Model.ContextWindow)
		}
	})
}

// Tests for CostOptimization

func TestCostOptimization(t *testing.T) {
	strategy := NewCostOptimization()

	t.Run("Name and Priority", func(t *testing.T) {
		assert.Equal(t, "cost_optimization", strategy.Name())
		assert.Equal(t, 50, strategy.Priority())
	})

	t.Run("Score without cost optimization does nothing", func(t *testing.T) {
		ctx := NewContext()
		ctx.Optimize = "quality"
		candidates := newTestCandidates()
		result := strategy.Score(ctx, candidates)

		for _, c := range result {
			assert.Equal(t, float64(0), c.Score)
		}
	})

	t.Run("Score with cost optimization prefers cheaper models", func(t *testing.T) {
		ctx := NewContext()
		ctx.Optimize = "cost"
		candidates := newTestCandidates()
		result := strategy.Score(ctx, candidates)

		// Find scores for different models
		var cheapestScore, expensiveScore float64
		for _, c := range result {
			if c.Model.ID == "gpt-4o-mini" { // Cheapest
				cheapestScore = c.Score
			}
			if c.Model.ID == "gpt-4o" { // Most expensive
				expensiveScore = c.Score
			}
		}
		assert.Greater(t, cheapestScore, expensiveScore)
	})
}

// Tests for LoadBalancing

func TestLoadBalancing(t *testing.T) {
	strategy := NewLoadBalancing()

	t.Run("Name and Priority", func(t *testing.T) {
		assert.Equal(t, "load_balancing", strategy.Name())
		assert.Equal(t, 10, strategy.Priority())
	})

	t.Run("Score adds jitter", func(t *testing.T) {
		ctx := NewContext()
		candidates := newTestCandidates()
		result := strategy.Score(ctx, candidates)

		for _, c := range result {
			assert.GreaterOrEqual(t, c.Score, float64(0))
			assert.Less(t, c.Score, float64(0.2))
		}
	})
}

// Tests for DefaultChain

func TestDefaultChain(t *testing.T) {
	t.Run("Creates chain with all strategies", func(t *testing.T) {
		chain := DefaultChain()
		assert.Equal(t, 6, len(chain.strategies))
	})

	t.Run("Execute with real candidates", func(t *testing.T) {
		chain := DefaultChain()
		ctx := NewContext()
		ctx.Optimize = "cost"
		ctx.ProviderHealth = make(map[string]bool)
		candidates := newTestCandidates()

		// Mark all providers healthy
		for _, c := range candidates {
			ctx.ProviderHealth[c.Provider.ID.String()] = true
		}

		result, err := chain.Execute(ctx, candidates)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Model)
		assert.NotEmpty(t, result.Reason)
	})
}

// Tests for ScoredCandidate

func TestScoredCandidate(t *testing.T) {
	t.Run("AddScore accumulates scores", func(t *testing.T) {
		p := newTestProvider("Provider")
		m := newTestModel("model", []string{"chat"}, 10000, 0.001, 0.002)
		c := NewScoredCandidate(p, m)

		c.AddScore("strategy1", 10, "reason1")
		c.AddScore("strategy2", 5, "reason2")

		assert.Equal(t, float64(15), c.Score)
		assert.Equal(t, float64(10), c.ScoreBreakdown["strategy1"])
		assert.Equal(t, float64(5), c.ScoreBreakdown["strategy2"])
		assert.Contains(t, c.Reasons, "reason1")
		assert.Contains(t, c.Reasons, "reason2")
	})
}

// Tests for Context

func TestContext(t *testing.T) {
	t.Run("NewContext creates defaults", func(t *testing.T) {
		ctx := NewContext()
		assert.Equal(t, "chat", ctx.TaskType)
		assert.NotNil(t, ctx.ProviderHealth)
		assert.NotNil(t, ctx.Metadata)
	})

	t.Run("RequiredCapabilities with no requirements", func(t *testing.T) {
		ctx := NewContext()
		caps := ctx.RequiredCapabilities()
		assert.Equal(t, 1, len(caps))
		assert.Equal(t, provider.CapabilityChat, caps[0])
	})

	t.Run("RequiredCapabilities with all requirements", func(t *testing.T) {
		ctx := NewContext()
		ctx.RequireStream = true
		ctx.RequireTools = true
		ctx.RequireVision = true
		ctx.RequireJSON = true

		caps := ctx.RequiredCapabilities()
		assert.Equal(t, 5, len(caps))
	})
}
