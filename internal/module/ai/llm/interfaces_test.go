package llm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/uniedit/server/internal/module/ai/adapter"
)

// mockChatService is a mock implementation of ChatService
type mockChatService struct {
	chatFunc       func(ctx context.Context, userID uuid.UUID, req *ChatRequest) (*ChatResponse, error)
	chatStreamFunc func(ctx context.Context, userID uuid.UUID, req *ChatRequest) (<-chan *adapter.ChatChunk, *RoutingInfo, error)
}

func (m *mockChatService) Chat(ctx context.Context, userID uuid.UUID, req *ChatRequest) (*ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *mockChatService) ChatStream(ctx context.Context, userID uuid.UUID, req *ChatRequest) (<-chan *adapter.ChatChunk, *RoutingInfo, error) {
	if m.chatStreamFunc != nil {
		return m.chatStreamFunc(ctx, userID, req)
	}
	return nil, nil, nil
}

// mockEmbeddingService is a mock implementation of EmbeddingService
type mockEmbeddingService struct {
	embedFunc func(ctx context.Context, userID uuid.UUID, req *EmbedRequest) (*adapter.EmbedResponse, error)
}

func (m *mockEmbeddingService) Embed(ctx context.Context, userID uuid.UUID, req *EmbedRequest) (*adapter.EmbedResponse, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, userID, req)
	}
	return nil, nil
}

func TestChatServiceInterface(t *testing.T) {
	// Ensure mockChatService implements ChatService
	var _ ChatService = (*mockChatService)(nil)
}

func TestEmbeddingServiceInterface(t *testing.T) {
	// Ensure mockEmbeddingService implements EmbeddingService
	var _ EmbeddingService = (*mockEmbeddingService)(nil)
}

func TestServiceImplementsLLMService(t *testing.T) {
	// Ensure Service implements LLMService
	var _ LLMService = (*Service)(nil)
}

func TestChatRequest(t *testing.T) {
	req := &ChatRequest{
		Model: "gpt-4o",
		Messages: []*adapter.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
		Stream:    false,
	}

	assert.Equal(t, "gpt-4o", req.Model)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, 100, req.MaxTokens)
	assert.False(t, req.Stream)
}

func TestEmbedRequest(t *testing.T) {
	req := &EmbedRequest{
		Model: "text-embedding-ada-002",
		Input: []string{"Hello", "World"},
	}

	assert.Equal(t, "text-embedding-ada-002", req.Model)
	assert.Len(t, req.Input, 2)
}

func TestRoutingInfo(t *testing.T) {
	info := &RoutingInfo{
		ProviderUsed: "openai",
		ModelUsed:    "gpt-4o",
		LatencyMs:    150,
		CostUSD:      0.001,
	}

	assert.Equal(t, "openai", info.ProviderUsed)
	assert.Equal(t, "gpt-4o", info.ModelUsed)
	assert.Equal(t, int64(150), info.LatencyMs)
	assert.Equal(t, 0.001, info.CostUSD)
}

func TestRoutingConfig(t *testing.T) {
	config := &RoutingConfig{
		Group:    "default",
		Strategy: "latency",
		Fallback: true,
	}

	assert.Equal(t, "default", config.Group)
	assert.Equal(t, "latency", config.Strategy)
	assert.True(t, config.Fallback)
}
