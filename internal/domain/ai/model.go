package ai

import (
	"time"

	"github.com/google/uuid"
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

// Model represents an AI model entity.
type Model struct {
	id              string
	providerID      uuid.UUID
	name            string
	capabilities    []Capability
	contextWindow   int
	maxOutputTokens int
	inputCostPer1K  float64
	outputCostPer1K float64
	enabled         bool
	options         map[string]any
	createdAt       time.Time
	updatedAt       time.Time
}

// NewModel creates a new model.
func NewModel(id string, providerID uuid.UUID, name string) *Model {
	return &Model{
		id:           id,
		providerID:   providerID,
		name:         name,
		capabilities: make([]Capability, 0),
		enabled:      true,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}
}

// ReconstructModel reconstructs a model from persistence.
func ReconstructModel(
	id string,
	providerID uuid.UUID,
	name string,
	capabilities []Capability,
	contextWindow int,
	maxOutputTokens int,
	inputCostPer1K float64,
	outputCostPer1K float64,
	enabled bool,
	options map[string]any,
	createdAt time.Time,
	updatedAt time.Time,
) *Model {
	return &Model{
		id:              id,
		providerID:      providerID,
		name:            name,
		capabilities:    capabilities,
		contextWindow:   contextWindow,
		maxOutputTokens: maxOutputTokens,
		inputCostPer1K:  inputCostPer1K,
		outputCostPer1K: outputCostPer1K,
		enabled:         enabled,
		options:         options,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

// Getters
func (m *Model) ID() string              { return m.id }
func (m *Model) ProviderID() uuid.UUID   { return m.providerID }
func (m *Model) Name() string            { return m.name }
func (m *Model) Capabilities() []Capability { return m.capabilities }
func (m *Model) ContextWindow() int      { return m.contextWindow }
func (m *Model) MaxOutputTokens() int    { return m.maxOutputTokens }
func (m *Model) InputCostPer1K() float64 { return m.inputCostPer1K }
func (m *Model) OutputCostPer1K() float64 { return m.outputCostPer1K }
func (m *Model) Enabled() bool           { return m.enabled }
func (m *Model) Options() map[string]any { return m.options }
func (m *Model) CreatedAt() time.Time    { return m.createdAt }
func (m *Model) UpdatedAt() time.Time    { return m.updatedAt }

// Setters
func (m *Model) SetName(name string)                 { m.name = name; m.updatedAt = time.Now() }
func (m *Model) SetCapabilities(caps []Capability)   { m.capabilities = caps; m.updatedAt = time.Now() }
func (m *Model) SetContextWindow(cw int)             { m.contextWindow = cw; m.updatedAt = time.Now() }
func (m *Model) SetMaxOutputTokens(max int)          { m.maxOutputTokens = max; m.updatedAt = time.Now() }
func (m *Model) SetInputCostPer1K(cost float64)      { m.inputCostPer1K = cost; m.updatedAt = time.Now() }
func (m *Model) SetOutputCostPer1K(cost float64)     { m.outputCostPer1K = cost; m.updatedAt = time.Now() }
func (m *Model) SetEnabled(enabled bool)             { m.enabled = enabled; m.updatedAt = time.Now() }
func (m *Model) SetOptions(opts map[string]any)      { m.options = opts; m.updatedAt = time.Now() }

// HasCapability checks if the model has a specific capability.
func (m *Model) HasCapability(cap Capability) bool {
	for _, c := range m.capabilities {
		if c == cap {
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
	return (m.inputCostPer1K + m.outputCostPer1K) / 2
}
