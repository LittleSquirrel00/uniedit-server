package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ===== Provider Types =====

// AIProviderType represents the type of AI provider.
type AIProviderType string

const (
	AIProviderTypeOpenAI    AIProviderType = "openai"
	AIProviderTypeAnthropic AIProviderType = "anthropic"
	AIProviderTypeGoogle    AIProviderType = "google"
	AIProviderTypeAzure     AIProviderType = "azure"
	AIProviderTypeOllama    AIProviderType = "ollama"
	AIProviderTypeGeneric   AIProviderType = "generic"
)

// AICapability represents a model capability.
type AICapability string

const (
	AICapabilityChat      AICapability = "chat"
	AICapabilityStream    AICapability = "stream"
	AICapabilityVision    AICapability = "vision"
	AICapabilityTools     AICapability = "tools"
	AICapabilityJSON      AICapability = "json_mode"
	AICapabilityEmbedding AICapability = "embedding"
	AICapabilityImage     AICapability = "image_generation"
	AICapabilityVideo     AICapability = "video_generation"
	AICapabilityAudio     AICapability = "audio_generation"
)

// ===== Entity Models =====

// AIRateLimitConfig defines rate limiting parameters.
type AIRateLimitConfig struct {
	RPM        int `json:"rpm"`         // Requests per minute
	TPM        int `json:"tpm"`         // Tokens per minute
	DailyLimit int `json:"daily_limit"` // Daily request limit
}

// AIProvider represents an AI provider configuration.
type AIProvider struct {
	ID        uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string             `json:"name" gorm:"not null"`
	Type      AIProviderType     `json:"type" gorm:"not null"`
	BaseURL   string             `json:"base_url" gorm:"column:base_url;not null"`
	APIKey    string             `json:"-" gorm:"column:api_key;not null"`
	Enabled   bool               `json:"enabled" gorm:"default:true"`
	Weight    int                `json:"weight" gorm:"default:1"`
	Priority  int                `json:"priority" gorm:"default:0"`
	RateLimit *AIRateLimitConfig `json:"rate_limit" gorm:"type:jsonb;serializer:json"`
	Options   map[string]any     `json:"options" gorm:"type:jsonb;serializer:json"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`

	// Relations
	Models []*AIModel `json:"models,omitempty" gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for AIProvider.
func (AIProvider) TableName() string {
	return "ai_providers"
}

// AIModel represents an AI model configuration.
type AIModel struct {
	ID              string         `json:"id" gorm:"primaryKey"`
	ProviderID      uuid.UUID      `json:"provider_id" gorm:"type:uuid;not null"`
	Name            string         `json:"name" gorm:"not null"`
	Capabilities    pq.StringArray `json:"capabilities" gorm:"type:text[];not null"`
	ContextWindow   int            `json:"context_window" gorm:"column:context_window;not null"`
	MaxOutputTokens int            `json:"max_output_tokens" gorm:"column:max_output_tokens;not null"`
	InputCostPer1K  float64        `json:"input_cost_per_1k" gorm:"column:input_cost_per_1k;type:decimal(10,6)"`
	OutputCostPer1K float64        `json:"output_cost_per_1k" gorm:"column:output_cost_per_1k;type:decimal(10,6)"`
	Enabled         bool           `json:"enabled" gorm:"default:true"`
	Options         map[string]any `json:"options" gorm:"type:jsonb;serializer:json"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`

	// Relations
	Provider *AIProvider `json:"provider,omitempty" gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for AIModel.
func (AIModel) TableName() string {
	return "ai_models"
}

// HasCapability checks if the model has a specific capability.
func (m *AIModel) HasCapability(cap AICapability) bool {
	for _, c := range m.Capabilities {
		if c == string(cap) {
			return true
		}
	}
	return false
}

// HasAllCapabilities checks if the model has all specified capabilities.
func (m *AIModel) HasAllCapabilities(caps []AICapability) bool {
	for _, cap := range caps {
		if !m.HasCapability(cap) {
			return false
		}
	}
	return true
}

// AverageCostPer1K calculates the average cost per 1K tokens.
func (m *AIModel) AverageCostPer1K() float64 {
	return (m.InputCostPer1K + m.OutputCostPer1K) / 2
}

// ===== Provider Account (Pool) =====

// AIHealthStatus represents the health state of a provider account.
type AIHealthStatus string

const (
	AIHealthStatusHealthy   AIHealthStatus = "healthy"
	AIHealthStatusDegraded  AIHealthStatus = "degraded"
	AIHealthStatusUnhealthy AIHealthStatus = "unhealthy"
)

// IsHealthy returns true if the status is healthy.
func (s AIHealthStatus) IsHealthy() bool {
	return s == AIHealthStatusHealthy
}

// CanServeRequests returns true if the account can serve requests.
func (s AIHealthStatus) CanServeRequests() bool {
	return s == AIHealthStatusHealthy || s == AIHealthStatusDegraded
}

// AIProviderAccount represents a single API key/account for a provider.
type AIProviderAccount struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProviderID      uuid.UUID `json:"provider_id" gorm:"type:uuid;not null;index"`
	Name            string    `json:"name" gorm:"not null"`
	EncryptedAPIKey string    `json:"-" gorm:"column:encrypted_api_key;not null"`
	KeyPrefix       string    `json:"key_prefix" gorm:"not null"`

	// Scheduling
	Weight   int  `json:"weight" gorm:"default:1"`
	Priority int  `json:"priority" gorm:"default:0"`
	IsActive bool `json:"is_active" gorm:"default:true"`

	// Health monitoring
	HealthStatus        AIHealthStatus `json:"health_status" gorm:"default:'healthy'"`
	LastHealthCheck     *time.Time     `json:"last_health_check,omitempty"`
	ConsecutiveFailures int            `json:"consecutive_failures" gorm:"default:0"`
	LastFailureAt       *time.Time     `json:"last_failure_at,omitempty"`

	// Rate limits (0 = use provider default)
	RateLimitRPM int `json:"rate_limit_rpm" gorm:"default:0"`
	RateLimitTPM int `json:"rate_limit_tpm" gorm:"default:0"`
	DailyLimit   int `json:"daily_limit" gorm:"default:0"`

	// Usage statistics (denormalized)
	TotalRequests int64   `json:"total_requests" gorm:"default:0"`
	TotalTokens   int64   `json:"total_tokens" gorm:"default:0"`
	TotalCostUSD  float64 `json:"total_cost_usd" gorm:"type:decimal(12,6);default:0"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Decrypted key (not stored, populated at runtime)
	DecryptedKey string `json:"-" gorm:"-"`
}

// TableName returns the database table name.
func (AIProviderAccount) TableName() string {
	return "provider_accounts"
}

// IsAvailable returns true if the account can handle requests.
func (a *AIProviderAccount) IsAvailable() bool {
	return a.IsActive && a.HealthStatus.CanServeRequests()
}

// AIAccountUsageStats represents daily usage statistics for an account.
type AIAccountUsageStats struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID     uuid.UUID `json:"account_id" gorm:"type:uuid;not null"`
	Date          time.Time `json:"date" gorm:"type:date;not null"`
	RequestsCount int64     `json:"requests_count" gorm:"default:0"`
	TokensCount   int64     `json:"tokens_count" gorm:"default:0"`
	CostUSD       float64   `json:"cost_usd" gorm:"type:decimal(12,6);default:0"`
	CreatedAt     time.Time `json:"created_at"`
}

// TableName returns the database table name.
func (AIAccountUsageStats) TableName() string {
	return "account_usage_stats"
}

// Health transition thresholds.
const (
	// AIFailuresToDegrade is the number of consecutive failures to transition from healthy to degraded.
	AIFailuresToDegrade = 2
	// AIFailuresToUnhealthy is the number of consecutive failures to transition from degraded to unhealthy.
	AIFailuresToUnhealthy = 5
	// AISuccessesToRecover is the number of consecutive successes to transition from degraded to healthy.
	AISuccessesToRecover = 3
	// AICircuitBreakerCooldown is the duration to wait before allowing a test request when unhealthy.
	AICircuitBreakerCooldown = 30 * time.Second
	// AILatencyThreshold is the latency threshold (in milliseconds) that triggers degradation.
	AILatencyThreshold = 3000
)

// ===== Model Group =====

// AISelectionStrategy defines how to select a model from the group.
type AISelectionStrategy string

const (
	AIStrategyPriority        AISelectionStrategy = "priority"
	AIStrategyRoundRobin      AISelectionStrategy = "round-robin"
	AIStrategyWeighted        AISelectionStrategy = "weighted"
	AIStrategyCostOptimal     AISelectionStrategy = "cost-optimal"
	AIStrategyQualityOptimal  AISelectionStrategy = "quality-optimal"
	AIStrategyLatencyOptimal  AISelectionStrategy = "latency-optimal"
	AIStrategyCapabilityMatch AISelectionStrategy = "capability-match"
)

// AITaskType defines the type of AI task.
type AITaskType string

const (
	AITaskTypeChat      AITaskType = "chat"
	AITaskTypeEmbedding AITaskType = "embedding"
	AITaskTypeImage     AITaskType = "image"
	AITaskTypeVideo     AITaskType = "video"
	AITaskTypeAudio     AITaskType = "audio"
)

// AIFallbackTrigger defines when to trigger fallback.
type AIFallbackTrigger string

const (
	AITriggerRateLimit   AIFallbackTrigger = "rate_limit"
	AITriggerTimeout     AIFallbackTrigger = "timeout"
	AITriggerServerError AIFallbackTrigger = "server_error"
)

// AIStrategyConfig contains strategy configuration.
type AIStrategyConfig struct {
	Type         AISelectionStrategy `json:"type"`
	Weights      map[string]int      `json:"weights,omitempty"`
	MaxCostPer1K float64             `json:"max_cost_per_1k,omitempty"`
}

// AIFallbackConfig contains fallback configuration.
type AIFallbackConfig struct {
	Enabled     bool                `json:"enabled"`
	MaxAttempts int                 `json:"max_attempts"`
	TriggerOn   []AIFallbackTrigger `json:"trigger_on"`
}

// AIModelGroup represents an AI model group configuration.
type AIModelGroup struct {
	ID                   string            `json:"id" gorm:"primaryKey"`
	Name                 string            `json:"name" gorm:"not null"`
	TaskType             AITaskType        `json:"task_type" gorm:"column:task_type;not null"`
	Models               pq.StringArray    `json:"models" gorm:"type:text[];not null"`
	Strategy             *AIStrategyConfig `json:"strategy" gorm:"type:jsonb;serializer:json;not null"`
	Fallback             *AIFallbackConfig `json:"fallback" gorm:"type:jsonb;serializer:json"`
	RequiredCapabilities pq.StringArray    `json:"required_capabilities" gorm:"type:text[]"`
	Enabled              bool              `json:"enabled" gorm:"default:true"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// TableName returns the table name for AIModelGroup.
func (AIModelGroup) TableName() string {
	return "ai_groups"
}

// HasModel checks if the group contains a specific model.
func (g *AIModelGroup) HasModel(modelID string) bool {
	for _, m := range g.Models {
		if m == modelID {
			return true
		}
	}
	return false
}

// ===== Request/Response Types =====

// AIChatRequest represents a chat completion request.
type AIChatRequest struct {
	Model       string           `json:"model"`
	Messages    []*AIChatMessage `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
	Tools       []*AITool        `json:"tools,omitempty"`
	ToolChoice  any              `json:"tool_choice,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
	UserID      uuid.UUID        `json:"-"` // Set by service layer
}

// AIChatMessage represents a chat message.
type AIChatMessage struct {
	Role       string        `json:"role"`
	Content    any           `json:"content"` // string or []AIContentPart
	Name       string        `json:"name,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	ToolCalls  []*AIToolCall `json:"tool_calls,omitempty"`
}

// GetTextContent extracts text content from a message.
func (m *AIChatMessage) GetTextContent() string {
	switch v := m.Content.(type) {
	case string:
		return v
	case []any:
		var text string
		for _, part := range v {
			if p, ok := part.(map[string]any); ok {
				if t, ok := p["type"].(string); ok && t == "text" {
					if txt, ok := p["text"].(string); ok {
						text += txt
					}
				}
			}
		}
		return text
	default:
		return ""
	}
}

// HasImages checks if the message contains images.
func (m *AIChatMessage) HasImages() bool {
	switch v := m.Content.(type) {
	case []any:
		for _, part := range v {
			if p, ok := part.(map[string]any); ok {
				if t, ok := p["type"].(string); ok && t == "image_url" {
					return true
				}
			}
		}
	}
	return false
}

// AIContentPart represents a multimodal content part.
type AIContentPart struct {
	Type     string      `json:"type"` // text, image_url
	Text     string      `json:"text,omitempty"`
	ImageURL *AIImageURL `json:"image_url,omitempty"`
}

// AIImageURL represents an image URL reference.
type AIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // auto, low, high
}

// AITool represents a tool definition.
type AITool struct {
	Type     string      `json:"type"` // function
	Function *AIFunction `json:"function"`
}

// AIFunction represents a function definition.
type AIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// AIToolCall represents a tool call in a response.
type AIToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"` // function
	Function *AIFunctionCall `json:"function"`
}

// AIFunctionCall represents a function call.
type AIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// AIChatResponse represents a chat completion response.
type AIChatResponse struct {
	ID           string         `json:"id"`
	Model        string         `json:"model"`
	Message      *AIChatMessage `json:"message"`
	FinishReason string         `json:"finish_reason"`
	Usage        *AIUsage       `json:"usage"`
	Routing      *AIRoutingInfo `json:"_routing,omitempty"`
}

// AIChatChunk represents a streaming chat chunk.
type AIChatChunk struct {
	ID           string   `json:"id"`
	Model        string   `json:"model"`
	Delta        *AIDelta `json:"delta"`
	FinishReason string   `json:"finish_reason,omitempty"`
}

// AIDelta represents incremental content.
type AIDelta struct {
	Role      string        `json:"role,omitempty"`
	Content   string        `json:"content,omitempty"`
	ToolCalls []*AIToolCall `json:"tool_calls,omitempty"`
}

// AIUsage represents token usage.
type AIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	// Prompt caching (vendor dependent). For OpenAI, CacheReadInputTokens is mapped from prompt_tokens_details.cached_tokens.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// AIRoutingInfo contains routing metadata.
type AIRoutingInfo struct {
	ProviderUsed string  `json:"provider_used"`
	ModelUsed    string  `json:"model_used"`
	LatencyMs    int64   `json:"latency_ms"`
	CostUSD      float64 `json:"cost_usd"`
}

// AIEmbedRequest represents an embedding request.
type AIEmbedRequest struct {
	Model  string    `json:"model"`
	Input  []string  `json:"input"`
	UserID uuid.UUID `json:"-"` // Set by service layer
}

// AIEmbedResponse represents an embedding response.
type AIEmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
	Usage      *AIUsage    `json:"usage,omitempty"`
}

// ===== Routing Types =====

// AIRoutingContext contains the context for routing decisions.
type AIRoutingContext struct {
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

// NewAIRoutingContext creates a new routing context with defaults.
func NewAIRoutingContext() *AIRoutingContext {
	return &AIRoutingContext{
		TaskType:       "chat",
		ProviderHealth: make(map[string]bool),
		Metadata:       make(map[string]any),
	}
}

// RequiredCapabilities returns the list of required capabilities.
func (c *AIRoutingContext) RequiredCapabilities() []AICapability {
	var caps []AICapability

	caps = append(caps, AICapabilityChat)

	if c.RequireStream {
		caps = append(caps, AICapabilityStream)
	}
	if c.RequireTools {
		caps = append(caps, AICapabilityTools)
	}
	if c.RequireVision {
		caps = append(caps, AICapabilityVision)
	}
	if c.RequireJSON {
		caps = append(caps, AICapabilityJSON)
	}

	return caps
}

// AIScoredCandidate represents a candidate with scoring information.
type AIScoredCandidate struct {
	Provider       *AIProvider
	Model          *AIModel
	Score          float64
	ScoreBreakdown map[string]float64
	Reasons        []string
}

// NewAIScoredCandidate creates a new scored candidate.
func NewAIScoredCandidate(p *AIProvider, m *AIModel) *AIScoredCandidate {
	return &AIScoredCandidate{
		Provider:       p,
		Model:          m,
		Score:          0,
		ScoreBreakdown: make(map[string]float64),
		Reasons:        make([]string, 0),
	}
}

// AddScore adds a score from a strategy.
func (c *AIScoredCandidate) AddScore(strategy string, score float64, reason string) {
	c.Score += score
	c.ScoreBreakdown[strategy] = score
	if reason != "" {
		c.Reasons = append(c.Reasons, reason)
	}
}

// AIRoutingResult represents the routing result.
type AIRoutingResult struct {
	Provider *AIProvider `json:"provider"`
	Model    *AIModel    `json:"model"`
	Score    float64     `json:"score"`
	Reason   string      `json:"reason"`

	// Account pool integration
	AccountID        *string `json:"account_id,omitempty"`         // Provider account ID if using pool
	AccountName      *string `json:"account_name,omitempty"`       // Provider account name if using pool
	AccountKeyPrefix *string `json:"account_key_prefix,omitempty"` // Provider account key prefix if using pool
	APIKey           string  `json:"-"`                            // Decrypted API key (from pool or provider)
}

// RequiresCapability checks if the request requires a specific capability.
func (r *AIChatRequest) RequiresCapability(cap AICapability) bool {
	switch cap {
	case AICapabilityStream:
		return r.Stream
	case AICapabilityTools:
		return len(r.Tools) > 0
	case AICapabilityVision:
		for _, msg := range r.Messages {
			if msg.HasImages() {
				return true
			}
		}
	}
	return false
}
