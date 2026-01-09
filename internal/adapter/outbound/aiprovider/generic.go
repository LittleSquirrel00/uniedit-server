package aiprovider

import (
	"context"
	"net/http"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// GenericAdapter implements the AIVendorAdapterPort interface for OpenAI-compatible APIs.
type GenericAdapter struct {
	*OpenAIAdapter
}

// NewGenericAdapter creates a new generic adapter with the given HTTP client.
func NewGenericAdapter(client *http.Client) *GenericAdapter {
	return &GenericAdapter{
		OpenAIAdapter: NewOpenAIAdapter(client),
	}
}

// Type returns the adapter type.
func (a *GenericAdapter) Type() model.AIProviderType {
	return model.AIProviderTypeGeneric
}

// Chat performs a non-streaming chat completion using OpenAI-compatible API.
func (a *GenericAdapter) Chat(ctx context.Context, req *model.AIChatRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIChatResponse, error) {
	return a.OpenAIAdapter.Chat(ctx, req, m, p, apiKey)
}

// ChatStream performs a streaming chat completion using OpenAI-compatible API.
func (a *GenericAdapter) ChatStream(ctx context.Context, req *model.AIChatRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (<-chan *model.AIChatChunk, error) {
	return a.OpenAIAdapter.ChatStream(ctx, req, m, p, apiKey)
}

// Embed generates text embeddings using OpenAI-compatible API.
func (a *GenericAdapter) Embed(ctx context.Context, req *model.AIEmbedRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIEmbedResponse, error) {
	return a.OpenAIAdapter.Embed(ctx, req, m, p, apiKey)
}

// HealthCheck performs a health check.
func (a *GenericAdapter) HealthCheck(ctx context.Context, provider *model.AIProvider, apiKey string) error {
	return a.OpenAIAdapter.HealthCheck(ctx, provider, apiKey)
}

// Compile-time interface assertions
var _ outbound.AIVendorAdapterPort = (*GenericAdapter)(nil)
