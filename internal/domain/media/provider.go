// Package media contains domain entities for media generation.
package media

import (
	"github.com/google/uuid"
)

// ProviderType represents the type of media provider.
type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAnthropic ProviderType = "anthropic"
	ProviderTypeGeneric   ProviderType = "generic"
)

// String returns the string representation of the provider type.
func (t ProviderType) String() string {
	return string(t)
}

// IsValid checks if the provider type is valid.
func (t ProviderType) IsValid() bool {
	switch t {
	case ProviderTypeOpenAI, ProviderTypeAnthropic, ProviderTypeGeneric:
		return true
	default:
		return false
	}
}

// Capability represents a media generation capability.
type Capability string

const (
	CapabilityImage Capability = "image"
	CapabilityVideo Capability = "video"
	CapabilityAudio Capability = "audio"
)

// String returns the string representation of the capability.
func (c Capability) String() string {
	return string(c)
}

// Provider represents a media provider configuration.
type Provider struct {
	id      uuid.UUID
	name    string
	ptype   ProviderType
	baseURL string
	apiKey  string
	enabled bool
}

// NewProvider creates a new media provider.
func NewProvider(name string, ptype ProviderType, baseURL, apiKey string) *Provider {
	return &Provider{
		id:      uuid.New(),
		name:    name,
		ptype:   ptype,
		baseURL: baseURL,
		apiKey:  apiKey,
		enabled: true,
	}
}

// ReconstructProvider reconstructs a provider from persistence.
func ReconstructProvider(id uuid.UUID, name string, ptype ProviderType, baseURL, apiKey string, enabled bool) *Provider {
	return &Provider{
		id:      id,
		name:    name,
		ptype:   ptype,
		baseURL: baseURL,
		apiKey:  apiKey,
		enabled: enabled,
	}
}

// ID returns the provider ID.
func (p *Provider) ID() uuid.UUID { return p.id }

// Name returns the provider name.
func (p *Provider) Name() string { return p.name }

// Type returns the provider type.
func (p *Provider) Type() ProviderType { return p.ptype }

// BaseURL returns the base URL.
func (p *Provider) BaseURL() string { return p.baseURL }

// APIKey returns the API key.
func (p *Provider) APIKey() string { return p.apiKey }

// Enabled returns whether the provider is enabled.
func (p *Provider) Enabled() bool { return p.enabled }

// Enable enables the provider.
func (p *Provider) Enable() {
	p.enabled = true
}

// Disable disables the provider.
func (p *Provider) Disable() {
	p.enabled = false
}

// UpdateAPIKey updates the API key.
func (p *Provider) UpdateAPIKey(apiKey string) {
	p.apiKey = apiKey
}

// Model represents a media generation model.
type Model struct {
	id           string
	providerID   uuid.UUID
	name         string
	capabilities []Capability
	enabled      bool
}

// NewModel creates a new media model.
func NewModel(id string, providerID uuid.UUID, name string, capabilities []Capability) *Model {
	return &Model{
		id:           id,
		providerID:   providerID,
		name:         name,
		capabilities: capabilities,
		enabled:      true,
	}
}

// ReconstructModel reconstructs a model from persistence.
func ReconstructModel(id string, providerID uuid.UUID, name string, capabilities []Capability, enabled bool) *Model {
	return &Model{
		id:           id,
		providerID:   providerID,
		name:         name,
		capabilities: capabilities,
		enabled:      enabled,
	}
}

// ID returns the model ID.
func (m *Model) ID() string { return m.id }

// ProviderID returns the provider ID.
func (m *Model) ProviderID() uuid.UUID { return m.providerID }

// Name returns the model name.
func (m *Model) Name() string { return m.name }

// Capabilities returns the model capabilities.
func (m *Model) Capabilities() []Capability { return m.capabilities }

// Enabled returns whether the model is enabled.
func (m *Model) Enabled() bool { return m.enabled }

// HasCapability checks if the model has a specific capability.
func (m *Model) HasCapability(cap Capability) bool {
	for _, c := range m.capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// Enable enables the model.
func (m *Model) Enable() {
	m.enabled = true
}

// Disable disables the model.
func (m *Model) Disable() {
	m.enabled = false
}
