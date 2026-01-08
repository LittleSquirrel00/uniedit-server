package ai

import (
	"time"

	"github.com/google/uuid"
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

// RateLimitConfig defines rate limiting parameters.
type RateLimitConfig struct {
	RPM        int `json:"rpm"`
	TPM        int `json:"tpm"`
	DailyLimit int `json:"daily_limit"`
}

// Provider represents an AI provider entity.
type Provider struct {
	id        uuid.UUID
	name      string
	pType     ProviderType
	baseURL   string
	apiKey    string
	enabled   bool
	weight    int
	priority  int
	rateLimit *RateLimitConfig
	options   map[string]any
	createdAt time.Time
	updatedAt time.Time
}

// NewProvider creates a new provider.
func NewProvider(name string, pType ProviderType, baseURL, apiKey string) *Provider {
	return &Provider{
		id:        uuid.New(),
		name:      name,
		pType:     pType,
		baseURL:   baseURL,
		apiKey:    apiKey,
		enabled:   true,
		weight:    1,
		priority:  0,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// ReconstructProvider reconstructs a provider from persistence.
func ReconstructProvider(
	id uuid.UUID,
	name string,
	pType ProviderType,
	baseURL string,
	apiKey string,
	enabled bool,
	weight int,
	priority int,
	rateLimit *RateLimitConfig,
	options map[string]any,
	createdAt time.Time,
	updatedAt time.Time,
) *Provider {
	return &Provider{
		id:        id,
		name:      name,
		pType:     pType,
		baseURL:   baseURL,
		apiKey:    apiKey,
		enabled:   enabled,
		weight:    weight,
		priority:  priority,
		rateLimit: rateLimit,
		options:   options,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// Getters
func (p *Provider) ID() uuid.UUID                { return p.id }
func (p *Provider) Name() string                 { return p.name }
func (p *Provider) Type() ProviderType           { return p.pType }
func (p *Provider) BaseURL() string              { return p.baseURL }
func (p *Provider) APIKey() string               { return p.apiKey }
func (p *Provider) Enabled() bool                { return p.enabled }
func (p *Provider) Weight() int                  { return p.weight }
func (p *Provider) Priority() int                { return p.priority }
func (p *Provider) RateLimit() *RateLimitConfig  { return p.rateLimit }
func (p *Provider) Options() map[string]any      { return p.options }
func (p *Provider) CreatedAt() time.Time         { return p.createdAt }
func (p *Provider) UpdatedAt() time.Time         { return p.updatedAt }

// Setters
func (p *Provider) SetName(name string)              { p.name = name; p.updatedAt = time.Now() }
func (p *Provider) SetBaseURL(url string)            { p.baseURL = url; p.updatedAt = time.Now() }
func (p *Provider) SetAPIKey(key string)             { p.apiKey = key; p.updatedAt = time.Now() }
func (p *Provider) SetEnabled(enabled bool)          { p.enabled = enabled; p.updatedAt = time.Now() }
func (p *Provider) SetWeight(weight int)             { p.weight = weight; p.updatedAt = time.Now() }
func (p *Provider) SetPriority(priority int)         { p.priority = priority; p.updatedAt = time.Now() }
func (p *Provider) SetRateLimit(rl *RateLimitConfig) { p.rateLimit = rl; p.updatedAt = time.Now() }
func (p *Provider) SetOptions(opts map[string]any)   { p.options = opts; p.updatedAt = time.Now() }

// IsHealthy returns true if the provider is enabled.
func (p *Provider) IsHealthy() bool {
	return p.enabled
}
