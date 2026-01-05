package adapter

import (
	"context"

	"github.com/uniedit/server/internal/module/ai/provider"
)

// Adapter defines the interface for LLM adapters.
type Adapter interface {
	// Type returns the adapter type identifier.
	Type() provider.ProviderType

	// Chat performs a non-streaming chat completion.
	Chat(ctx context.Context, req *ChatRequest, model *provider.Model, prov *provider.Provider) (*ChatResponse, error)

	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, req *ChatRequest, model *provider.Model, prov *provider.Provider) (<-chan *ChatChunk, error)

	// Embed generates text embeddings.
	Embed(ctx context.Context, input []string, model *provider.Model, prov *provider.Provider) (*EmbedResponse, error)

	// HealthCheck performs a health check.
	HealthCheck(ctx context.Context, prov *provider.Provider) error

	// SupportsCapability checks if the adapter supports a capability.
	SupportsCapability(cap provider.Capability) bool
}

// BaseAdapter provides common functionality for adapters.
type BaseAdapter struct {
	capabilities map[provider.Capability]bool
}

// NewBaseAdapter creates a new base adapter.
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
