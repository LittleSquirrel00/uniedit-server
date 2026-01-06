package routing

import (
	"github.com/uniedit/server/internal/module/ai/provider"
)

// Context contains the context for routing decisions.
type Context struct {
	// Task type (chat, embedding, image, video)
	TaskType string

	// Token estimation
	EstimatedTokens int

	// Required capabilities
	RequireStream bool
	RequireTools  bool
	RequireVision bool
	RequireJSON   bool

	// Context window requirements
	MinContextWindow int

	// Cost constraints
	MaxCostPer1K float64

	// Optimization preference (cost, quality, speed)
	Optimize string

	// Provider preferences
	PreferredProviders []string
	ExcludedProviders  []string
	PreferredModels    []string

	// Health status (injected by routing manager)
	ProviderHealth map[string]bool

	// Group override
	GroupID string

	// Additional metadata
	Metadata map[string]any
}

// NewContext creates a new routing context with defaults.
func NewContext() *Context {
	return &Context{
		TaskType:       "chat",
		ProviderHealth: make(map[string]bool),
		Metadata:       make(map[string]any),
	}
}

// RequiredCapabilities returns the list of required capabilities.
func (c *Context) RequiredCapabilities() []provider.Capability {
	var caps []provider.Capability

	caps = append(caps, provider.CapabilityChat)

	if c.RequireStream {
		caps = append(caps, provider.CapabilityStream)
	}
	if c.RequireTools {
		caps = append(caps, provider.CapabilityTools)
	}
	if c.RequireVision {
		caps = append(caps, provider.CapabilityVision)
	}
	if c.RequireJSON {
		caps = append(caps, provider.CapabilityJSON)
	}

	return caps
}

// ScoredCandidate represents a candidate with scoring information.
type ScoredCandidate struct {
	Provider       *provider.Provider
	Model          *provider.Model
	Score          float64
	ScoreBreakdown map[string]float64
	Reasons        []string
}

// NewScoredCandidate creates a new scored candidate.
func NewScoredCandidate(p *provider.Provider, m *provider.Model) *ScoredCandidate {
	return &ScoredCandidate{
		Provider:       p,
		Model:          m,
		Score:          0,
		ScoreBreakdown: make(map[string]float64),
		Reasons:        make([]string, 0),
	}
}

// AddScore adds a score from a strategy.
func (c *ScoredCandidate) AddScore(strategy string, score float64, reason string) {
	c.Score += score
	c.ScoreBreakdown[strategy] = score
	if reason != "" {
		c.Reasons = append(c.Reasons, reason)
	}
}

// Result represents the routing result.
type Result struct {
	Provider *provider.Provider `json:"provider"`
	Model    *provider.Model    `json:"model"`
	Score    float64            `json:"score"`
	Reason   string             `json:"reason"`

	// Account pool integration
	AccountID *string `json:"account_id,omitempty"` // Provider account ID if using pool
	APIKey    string  `json:"-"`                    // Decrypted API key (from pool or provider)
}
