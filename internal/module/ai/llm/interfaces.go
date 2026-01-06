package llm

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/ai/adapter"
)

// ChatService provides chat completion operations.
type ChatService interface {
	// Chat performs a non-streaming chat completion.
	Chat(ctx context.Context, userID uuid.UUID, req *ChatRequest) (*ChatResponse, error)

	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, userID uuid.UUID, req *ChatRequest) (<-chan *adapter.ChatChunk, *RoutingInfo, error)
}

// EmbeddingService provides text embedding operations.
type EmbeddingService interface {
	// Embed generates text embeddings.
	Embed(ctx context.Context, userID uuid.UUID, req *EmbedRequest) (*adapter.EmbedResponse, error)
}

// LLMService combines all LLM operations.
// This interface is for backward compatibility.
type LLMService interface {
	ChatService
	EmbeddingService
}

// Compile-time interface assertions
var _ LLMService = (*Service)(nil)
