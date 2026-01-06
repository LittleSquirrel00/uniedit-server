package adapter

import (
	"context"

	"github.com/uniedit/server/internal/module/ai/provider"
)

// GenericAdapter implements the Adapter interface for OpenAI-compatible APIs.
type GenericAdapter struct {
	*OpenAIAdapter
}

// NewGenericAdapter creates a new generic adapter.
func NewGenericAdapter() *GenericAdapter {
	return &GenericAdapter{
		OpenAIAdapter: NewOpenAIAdapter(),
	}
}

// Type returns the adapter type.
func (a *GenericAdapter) Type() provider.ProviderType {
	return provider.ProviderTypeGeneric
}

// Chat performs a non-streaming chat completion using OpenAI-compatible API.
func (a *GenericAdapter) Chat(ctx context.Context, req *ChatRequest, model *provider.Model, prov *provider.Provider) (*ChatResponse, error) {
	return a.OpenAIAdapter.Chat(ctx, req, model, prov)
}

// ChatStream performs a streaming chat completion using OpenAI-compatible API.
func (a *GenericAdapter) ChatStream(ctx context.Context, req *ChatRequest, model *provider.Model, prov *provider.Provider) (<-chan *ChatChunk, error) {
	return a.OpenAIAdapter.ChatStream(ctx, req, model, prov)
}

// Embed generates text embeddings using OpenAI-compatible API.
func (a *GenericAdapter) Embed(ctx context.Context, input []string, model *provider.Model, prov *provider.Provider) (*EmbedResponse, error) {
	return a.OpenAIAdapter.Embed(ctx, input, model, prov)
}

// HealthCheck performs a health check.
func (a *GenericAdapter) HealthCheck(ctx context.Context, prov *provider.Provider) error {
	return a.OpenAIAdapter.HealthCheck(ctx, prov)
}

// Compile-time interface assertions
var (
	_ Adapter          = (*GenericAdapter)(nil)
	_ TypedAdapter     = (*GenericAdapter)(nil)
	_ TextAdapter      = (*GenericAdapter)(nil)
	_ EmbeddingAdapter = (*GenericAdapter)(nil)
	_ HealthChecker    = (*GenericAdapter)(nil)
)
