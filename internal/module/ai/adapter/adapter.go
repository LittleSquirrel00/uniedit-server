package adapter

import (
	"context"

	"github.com/uniedit/server/internal/module/ai/provider"
)

// TypedAdapter provides type identification for adapters.
type TypedAdapter interface {
	// Type returns the adapter type identifier.
	Type() provider.ProviderType
}

// CapabilityChecker checks adapter capabilities.
type CapabilityChecker interface {
	// SupportsCapability checks if the adapter supports a capability.
	SupportsCapability(cap provider.Capability) bool
}

// HealthChecker performs health checks on providers.
type HealthChecker interface {
	// HealthCheck performs a health check on a provider.
	HealthCheck(ctx context.Context, prov *provider.Provider) error
}

// TextAdapter handles text generation (chat completions).
type TextAdapter interface {
	// Chat performs a non-streaming chat completion.
	Chat(ctx context.Context, req *ChatRequest, model *provider.Model, prov *provider.Provider) (*ChatResponse, error)

	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, req *ChatRequest, model *provider.Model, prov *provider.Provider) (<-chan *ChatChunk, error)
}

// EmbeddingAdapter handles text embeddings.
type EmbeddingAdapter interface {
	// Embed generates text embeddings.
	Embed(ctx context.Context, input []string, model *provider.Model, prov *provider.Provider) (*EmbedResponse, error)
}

// Adapter is the unified interface that combines all adapter capabilities.
// This is kept for backward compatibility; new code should depend on specific interfaces.
type Adapter interface {
	TypedAdapter
	CapabilityChecker
	HealthChecker
	TextAdapter
	EmbeddingAdapter
}

// BaseAdapter provides common functionality for adapters.
type BaseAdapter struct {
	capabilities map[provider.Capability]bool
}

// NewBaseAdapter creates a new base adapter with specified capabilities.
func NewBaseAdapter(caps ...provider.Capability) *BaseAdapter {
	capMap := make(map[provider.Capability]bool)
	for _, cap := range caps {
		capMap[cap] = true
	}
	return &BaseAdapter{capabilities: capMap}
}

// SupportsCapability checks if the adapter supports a capability.
func (b *BaseAdapter) SupportsCapability(cap provider.Capability) bool {
	return b.capabilities[cap]
}

// AddCapability adds a capability to the adapter.
func (b *BaseAdapter) AddCapability(cap provider.Capability) {
	b.capabilities[cap] = true
}

// RemoveCapability removes a capability from the adapter.
func (b *BaseAdapter) RemoveCapability(cap provider.Capability) {
	delete(b.capabilities, cap)
}

// Capabilities returns all supported capabilities.
func (b *BaseAdapter) Capabilities() []provider.Capability {
	caps := make([]provider.Capability, 0, len(b.capabilities))
	for cap := range b.capabilities {
		caps = append(caps, cap)
	}
	return caps
}

// Compile-time interface assertions
var (
	_ CapabilityChecker = (*BaseAdapter)(nil)
)
