package provider

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ProviderType represents the type of AI provider.
type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAnthropic ProviderType = "anthropic"
	ProviderTypeGoogle    ProviderType = "google"
	ProviderTypeAzure     ProviderType = "azure"
	ProviderTypeOllama    ProviderType = "ollama"
	ProviderTypeGeneric   ProviderType = "generic"
)

// Capability represents a model capability.
type Capability string

const (
	CapabilityChat      Capability = "chat"
	CapabilityStream    Capability = "stream"
	CapabilityVision    Capability = "vision"
	CapabilityTools     Capability = "tools"
	CapabilityJSON      Capability = "json_mode"
	CapabilityEmbedding Capability = "embedding"
	CapabilityImage     Capability = "image_generation"
	CapabilityVideo     Capability = "video_generation"
	CapabilityAudio     Capability = "audio_generation"
)

// RateLimitConfig defines rate limiting parameters.
type RateLimitConfig struct {
	RPM        int `json:"rpm"`         // Requests per minute
	TPM        int `json:"tpm"`         // Tokens per minute
	DailyLimit int `json:"daily_limit"` // Daily request limit
}

// Provider represents an AI provider configuration.
type Provider struct {
	ID        uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string           `json:"name" gorm:"not null"`
	Type      ProviderType     `json:"type" gorm:"not null"`
	BaseURL   string           `json:"base_url" gorm:"column:base_url;not null"`
	APIKey    string           `json:"-" gorm:"column:api_key;not null"`
	Enabled   bool             `json:"enabled" gorm:"default:true"`
	Weight    int              `json:"weight" gorm:"default:1"`
	Priority  int              `json:"priority" gorm:"default:0"`
	RateLimit *RateLimitConfig `json:"rate_limit" gorm:"type:jsonb;serializer:json"`
	Options   map[string]any   `json:"options" gorm:"type:jsonb;serializer:json"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`

	// Relations
	Models []*Model `json:"models,omitempty" gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for Provider.
func (Provider) TableName() string {
	return "ai_providers"
}

// Model represents an AI model configuration.
type Model struct {
	ID              string         `json:"id" gorm:"primaryKey"`
	ProviderID      uuid.UUID      `json:"provider_id" gorm:"type:uuid;not null"`
	Name            string         `json:"name" gorm:"not null"`
	Capabilities    pq.StringArray `json:"capabilities" gorm:"type:text[];not null"`
	ContextWindow   int            `json:"context_window" gorm:"not null"`
	MaxOutputTokens int            `json:"max_output_tokens" gorm:"not null"`
	InputCostPer1K  float64        `json:"input_cost_per_1k" gorm:"type:decimal(10,6)"`
	OutputCostPer1K float64        `json:"output_cost_per_1k" gorm:"type:decimal(10,6)"`
	Enabled         bool           `json:"enabled" gorm:"default:true"`
	Options         map[string]any `json:"options" gorm:"type:jsonb;serializer:json"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`

	// Relations
	Provider *Provider `json:"provider,omitempty" gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for Model.
func (Model) TableName() string {
	return "ai_models"
}

// HasCapability checks if the model has a specific capability.
func (m *Model) HasCapability(cap Capability) bool {
	for _, c := range m.Capabilities {
		if c == string(cap) {
			return true
		}
	}
	return false
}

// HasAllCapabilities checks if the model has all specified capabilities.
func (m *Model) HasAllCapabilities(caps []Capability) bool {
	for _, cap := range caps {
		if !m.HasCapability(cap) {
			return false
		}
	}
	return true
}

// AverageCostPer1K calculates the average cost per 1K tokens.
func (m *Model) AverageCostPer1K() float64 {
	return (m.InputCostPer1K + m.OutputCostPer1K) / 2
}
