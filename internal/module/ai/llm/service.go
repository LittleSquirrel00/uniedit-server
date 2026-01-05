package llm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/ai/adapter"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/ai/routing"
)

// Service provides LLM operations.
type Service struct {
	registry        *provider.Registry
	healthMonitor   *provider.HealthMonitor
	routingManager  *routing.Manager
	adapterRegistry *adapter.Registry
}

// NewService creates a new LLM service.
func NewService(
	registry *provider.Registry,
	healthMonitor *provider.HealthMonitor,
	routingManager *routing.Manager,
) *Service {
	return &Service{
		registry:        registry,
		healthMonitor:   healthMonitor,
		routingManager:  routingManager,
		adapterRegistry: adapter.GetRegistry(),
	}
}

// ChatRequest represents a chat request from the API.
type ChatRequest struct {
	Model       string             `json:"model"`
	Messages    []*adapter.Message `json:"messages"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	TopP        *float64           `json:"top_p,omitempty"`
	Stop        []string           `json:"stop,omitempty"`
	Tools       []*adapter.Tool    `json:"tools,omitempty"`
	ToolChoice  any                `json:"tool_choice,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Routing     *RoutingConfig     `json:"routing,omitempty"`
}

// RoutingConfig contains routing configuration.
type RoutingConfig struct {
	Group    string `json:"group,omitempty"`
	Strategy string `json:"strategy,omitempty"`
	Fallback bool   `json:"fallback,omitempty"`
}

// ChatResponse represents a chat response.
type ChatResponse struct {
	*adapter.ChatResponse
	Routing *RoutingInfo `json:"_routing,omitempty"`
}

// RoutingInfo contains routing metadata.
type RoutingInfo struct {
	ProviderUsed string  `json:"provider_used"`
	ModelUsed    string  `json:"model_used"`
	LatencyMs    int64   `json:"latency_ms"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}

// Chat performs a non-streaming chat completion.
func (s *Service) Chat(ctx context.Context, userID uuid.UUID, req *ChatRequest) (*ChatResponse, error) {
	// Build routing context
	routingCtx := s.buildRoutingContext(req)

	// Route to best model
	result, err := s.routingManager.Route(ctx, routingCtx)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	// Get adapter
	adpt, err := s.adapterRegistry.GetForProvider(result.Provider)
	if err != nil {
		return nil, fmt.Errorf("get adapter: %w", err)
	}

	// Build adapter request
	adapterReq := &adapter.ChatRequest{
		Model:       result.Model.ID,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
		Tools:       req.Tools,
		ToolChoice:  req.ToolChoice,
		Stream:      false,
	}

	// Execute
	resp, err := adpt.Chat(ctx, adapterReq, result.Model, result.Provider)
	if err != nil {
		return nil, fmt.Errorf("chat failed: %w", err)
	}

	return &ChatResponse{
		ChatResponse: resp,
		Routing: &RoutingInfo{
			ProviderUsed: result.Provider.Name,
			ModelUsed:    result.Model.ID,
		},
	}, nil
}

// ChatStream performs a streaming chat completion.
func (s *Service) ChatStream(ctx context.Context, userID uuid.UUID, req *ChatRequest) (<-chan *adapter.ChatChunk, *RoutingInfo, error) {
	// Build routing context
	routingCtx := s.buildRoutingContext(req)
	routingCtx.RequireStream = true

	// Route to best model
	result, err := s.routingManager.Route(ctx, routingCtx)
	if err != nil {
		return nil, nil, fmt.Errorf("routing failed: %w", err)
	}

	// Get adapter
	adpt, err := s.adapterRegistry.GetForProvider(result.Provider)
	if err != nil {
		return nil, nil, fmt.Errorf("get adapter: %w", err)
	}

	// Build adapter request
	adapterReq := &adapter.ChatRequest{
		Model:       result.Model.ID,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
		Tools:       req.Tools,
		ToolChoice:  req.ToolChoice,
		Stream:      true,
	}

	// Execute
	chunks, err := adpt.ChatStream(ctx, adapterReq, result.Model, result.Provider)
	if err != nil {
		return nil, nil, fmt.Errorf("chat stream failed: %w", err)
	}

	routingInfo := &RoutingInfo{
		ProviderUsed: result.Provider.Name,
		ModelUsed:    result.Model.ID,
	}

	return chunks, routingInfo, nil
}

// EmbedRequest represents an embedding request.
type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// Embed generates text embeddings.
func (s *Service) Embed(ctx context.Context, userID uuid.UUID, req *EmbedRequest) (*adapter.EmbedResponse, error) {
	// Build routing context
	routingCtx := routing.NewContext()
	routingCtx.TaskType = "embedding"

	// If model specified, use it directly
	if req.Model != "" && req.Model != "auto" {
		model, prov, ok := s.registry.GetModelWithProvider(req.Model)
		if !ok {
			return nil, fmt.Errorf("model not found: %s", req.Model)
		}

		adpt, err := s.adapterRegistry.GetForProvider(prov)
		if err != nil {
			return nil, fmt.Errorf("get adapter: %w", err)
		}

		return adpt.Embed(ctx, req.Input, model, prov)
	}

	// Route to best model
	result, err := s.routingManager.Route(ctx, routingCtx)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	// Get adapter
	adpt, err := s.adapterRegistry.GetForProvider(result.Provider)
	if err != nil {
		return nil, fmt.Errorf("get adapter: %w", err)
	}

	return adpt.Embed(ctx, req.Input, result.Model, result.Provider)
}

// buildRoutingContext builds a routing context from a chat request.
func (s *Service) buildRoutingContext(req *ChatRequest) *routing.Context {
	ctx := routing.NewContext()
	ctx.TaskType = "chat"
	ctx.RequireStream = req.Stream

	// Detect required capabilities from messages
	for _, msg := range req.Messages {
		if msg.HasImages() {
			ctx.RequireVision = true
			break
		}
	}

	if len(req.Tools) > 0 {
		ctx.RequireTools = true
	}

	// Apply routing config
	if req.Routing != nil {
		ctx.GroupID = req.Routing.Group
		ctx.Optimize = req.Routing.Strategy
	}

	// If specific model requested
	if req.Model != "" && req.Model != "auto" {
		ctx.PreferredModels = []string{req.Model}
	}

	return ctx
}
