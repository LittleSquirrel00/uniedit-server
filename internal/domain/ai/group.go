package ai

import (
	"time"
)

// SelectionStrategy defines how to select a model from the group.
type SelectionStrategy string

const (
	StrategyPriority        SelectionStrategy = "priority"
	StrategyRoundRobin      SelectionStrategy = "round-robin"
	StrategyWeighted        SelectionStrategy = "weighted"
	StrategyCostOptimal     SelectionStrategy = "cost-optimal"
	StrategyQualityOptimal  SelectionStrategy = "quality-optimal"
	StrategyLatencyOptimal  SelectionStrategy = "latency-optimal"
	StrategyCapabilityMatch SelectionStrategy = "capability-match"
)

// TaskType defines the type of AI task.
type TaskType string

const (
	TaskTypeChat      TaskType = "chat"
	TaskTypeEmbedding TaskType = "embedding"
	TaskTypeImage     TaskType = "image"
	TaskTypeVideo     TaskType = "video"
	TaskTypeAudio     TaskType = "audio"
)

// FallbackTrigger defines when to trigger fallback.
type FallbackTrigger string

const (
	TriggerRateLimit   FallbackTrigger = "rate_limit"
	TriggerTimeout     FallbackTrigger = "timeout"
	TriggerServerError FallbackTrigger = "server_error"
)

// StrategyConfig contains strategy configuration.
type StrategyConfig struct {
	Type         SelectionStrategy `json:"type"`
	Weights      map[string]int    `json:"weights,omitempty"`
	MaxCostPer1K float64           `json:"max_cost_per_1k,omitempty"`
}

// FallbackConfig contains fallback configuration.
type FallbackConfig struct {
	Enabled     bool              `json:"enabled"`
	MaxAttempts int               `json:"max_attempts"`
	TriggerOn   []FallbackTrigger `json:"trigger_on"`
}

// Group represents an AI model group entity.
type Group struct {
	id                   string
	name                 string
	taskType             TaskType
	models               []string
	strategy             *StrategyConfig
	fallback             *FallbackConfig
	requiredCapabilities []Capability
	enabled              bool
	createdAt            time.Time
	updatedAt            time.Time
}

// NewGroup creates a new group.
func NewGroup(id, name string, taskType TaskType, models []string) *Group {
	return &Group{
		id:        id,
		name:      name,
		taskType:  taskType,
		models:    models,
		strategy:  &StrategyConfig{Type: StrategyPriority},
		enabled:   true,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// ReconstructGroup reconstructs a group from persistence.
func ReconstructGroup(
	id string,
	name string,
	taskType TaskType,
	models []string,
	strategy *StrategyConfig,
	fallback *FallbackConfig,
	requiredCapabilities []Capability,
	enabled bool,
	createdAt time.Time,
	updatedAt time.Time,
) *Group {
	return &Group{
		id:                   id,
		name:                 name,
		taskType:             taskType,
		models:               models,
		strategy:             strategy,
		fallback:             fallback,
		requiredCapabilities: requiredCapabilities,
		enabled:              enabled,
		createdAt:            createdAt,
		updatedAt:            updatedAt,
	}
}

// Getters
func (g *Group) ID() string                         { return g.id }
func (g *Group) Name() string                       { return g.name }
func (g *Group) TaskType() TaskType                 { return g.taskType }
func (g *Group) Models() []string                   { return g.models }
func (g *Group) Strategy() *StrategyConfig          { return g.strategy }
func (g *Group) Fallback() *FallbackConfig          { return g.fallback }
func (g *Group) RequiredCapabilities() []Capability { return g.requiredCapabilities }
func (g *Group) Enabled() bool                      { return g.enabled }
func (g *Group) CreatedAt() time.Time               { return g.createdAt }
func (g *Group) UpdatedAt() time.Time               { return g.updatedAt }

// Setters
func (g *Group) SetName(name string)                            { g.name = name; g.updatedAt = time.Now() }
func (g *Group) SetModels(models []string)                      { g.models = models; g.updatedAt = time.Now() }
func (g *Group) SetStrategy(s *StrategyConfig)                  { g.strategy = s; g.updatedAt = time.Now() }
func (g *Group) SetFallback(f *FallbackConfig)                  { g.fallback = f; g.updatedAt = time.Now() }
func (g *Group) SetRequiredCapabilities(caps []Capability)      { g.requiredCapabilities = caps; g.updatedAt = time.Now() }
func (g *Group) SetEnabled(enabled bool)                        { g.enabled = enabled; g.updatedAt = time.Now() }

// HasModel checks if the group contains a specific model.
func (g *Group) HasModel(modelID string) bool {
	for _, m := range g.models {
		if m == modelID {
			return true
		}
	}
	return false
}

// AddModel adds a model to the group.
func (g *Group) AddModel(modelID string) {
	if !g.HasModel(modelID) {
		g.models = append(g.models, modelID)
		g.updatedAt = time.Now()
	}
}

// RemoveModel removes a model from the group.
func (g *Group) RemoveModel(modelID string) {
	for i, m := range g.models {
		if m == modelID {
			g.models = append(g.models[:i], g.models[i+1:]...)
			g.updatedAt = time.Now()
			return
		}
	}
}
