package group

import (
	"time"

	"github.com/lib/pq"
)

// SelectionStrategy defines how to select a model from the group.
type SelectionStrategy string

const (
	StrategyPriority       SelectionStrategy = "priority"
	StrategyRoundRobin     SelectionStrategy = "round-robin"
	StrategyWeighted       SelectionStrategy = "weighted"
	StrategyCostOptimal    SelectionStrategy = "cost-optimal"
	StrategyQualityOptimal SelectionStrategy = "quality-optimal"
	StrategyLatencyOptimal SelectionStrategy = "latency-optimal"
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

// Group represents an AI model group configuration.
type Group struct {
	ID                   string          `json:"id" gorm:"primaryKey"`
	Name                 string          `json:"name" gorm:"not null"`
	TaskType             TaskType        `json:"task_type" gorm:"column:task_type;not null"`
	Models               pq.StringArray  `json:"models" gorm:"type:text[];not null"`
	Strategy             *StrategyConfig `json:"strategy" gorm:"type:jsonb;serializer:json;not null"`
	Fallback             *FallbackConfig `json:"fallback" gorm:"type:jsonb;serializer:json"`
	RequiredCapabilities pq.StringArray  `json:"required_capabilities" gorm:"type:text[]"`
	Enabled              bool            `json:"enabled" gorm:"default:true"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// TableName returns the table name for Group.
func (Group) TableName() string {
	return "ai_groups"
}

// HasModel checks if the group contains a specific model.
func (g *Group) HasModel(modelID string) bool {
	for _, m := range g.Models {
		if m == modelID {
			return true
		}
	}
	return false
}
