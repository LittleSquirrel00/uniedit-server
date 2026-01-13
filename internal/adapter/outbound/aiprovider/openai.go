package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// OpenAIAdapter implements the AIVendorAdapterPort interface for OpenAI.
type OpenAIAdapter struct {
	*BaseAdapter
	client *http.Client
}

// NewOpenAIAdapter creates a new OpenAI adapter with the given HTTP client.
func NewOpenAIAdapter(client *http.Client) *OpenAIAdapter {
	return &OpenAIAdapter{
		BaseAdapter: NewBaseAdapter(
			model.AICapabilityChat,
			model.AICapabilityStream,
			model.AICapabilityVision,
			model.AICapabilityTools,
			model.AICapabilityJSON,
			model.AICapabilityEmbedding,
		),
		client: client,
	}
}

// Type returns the adapter type.
func (a *OpenAIAdapter) Type() model.AIProviderType {
	return model.AIProviderTypeOpenAI
}

// HealthCheck performs a health check on the provider.
func (a *OpenAIAdapter) HealthCheck(ctx context.Context, provider *model.AIProvider, apiKey string) error {
	// Simple models list request to check connectivity
	req, err := http.NewRequestWithContext(ctx, "GET", provider.BaseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// Chat performs a non-streaming chat completion.
func (a *OpenAIAdapter) Chat(ctx context.Context, req *model.AIChatRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIChatResponse, error) {
	// Build request body
	body := a.buildChatRequest(req, m)

	// Make API request
	respBody, err := a.doRequest(ctx, p, apiKey, "/chat/completions", body)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	// Parse response
	type openaiUsage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details,omitempty"`
	}
	var openaiResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Index        int                  `json:"index"`
			Message      *model.AIChatMessage `json:"message"`
			FinishReason string               `json:"finish_reason"`
		} `json:"choices"`
		Usage *openaiUsage `json:"usage"`
	}

	if err := json.NewDecoder(respBody).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	var usage *model.AIUsage
	if openaiResp.Usage != nil {
		usage = &model.AIUsage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		}
		if openaiResp.Usage.PromptTokensDetails != nil {
			usage.CacheReadInputTokens = openaiResp.Usage.PromptTokensDetails.CachedTokens
		}
	}

	return &model.AIChatResponse{
		ID:           openaiResp.ID,
		Model:        openaiResp.Model,
		Message:      openaiResp.Choices[0].Message,
		FinishReason: openaiResp.Choices[0].FinishReason,
		Usage:        usage,
	}, nil
}

// ChatStream performs a streaming chat completion.
func (a *OpenAIAdapter) ChatStream(ctx context.Context, req *model.AIChatRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (<-chan *model.AIChatChunk, error) {
	// Build request body with streaming enabled
	body := a.buildChatRequest(req, m)
	body["stream"] = true
	body["stream_options"] = map[string]any{
		"include_usage": true,
	}

	// Make API request
	respBody, err := a.doRequest(ctx, p, apiKey, "/chat/completions", body)
	if err != nil {
		return nil, err
	}

	// Create channel for chunks
	chunks := make(chan *model.AIChatChunk, 100)

	// Parse SSE stream in goroutine
	go func() {
		defer close(chunks)
		defer respBody.Close()

		parser := NewSSEParser(respBody)
		for {
			event, err := parser.Next()
			if err != nil {
				if err == io.EOF {
					return
				}
				// Log error and return
				return
			}

			chunk, err := ParseOpenAIChunk(event.Data)
			if err != nil {
				if err == io.EOF {
					return
				}
				continue
			}

			select {
			case <-ctx.Done():
				return
			case chunks <- chunk:
			}
		}
	}()

	return chunks, nil
}

// Embed generates text embeddings.
func (a *OpenAIAdapter) Embed(ctx context.Context, req *model.AIEmbedRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIEmbedResponse, error) {
	body := map[string]any{
		"model": m.ID,
		"input": req.Input,
	}

	respBody, err := a.doRequest(ctx, p, apiKey, "/embeddings", body)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	type openaiUsage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details,omitempty"`
	}
	var openaiResp struct {
		Model string `json:"model"`
		Data  []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Usage *openaiUsage `json:"usage"`
	}

	if err := json.NewDecoder(respBody).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	embeddings := make([][]float64, len(openaiResp.Data))
	for _, d := range openaiResp.Data {
		embeddings[d.Index] = d.Embedding
	}

	var usage *model.AIUsage
	if openaiResp.Usage != nil {
		usage = &model.AIUsage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		}
		if openaiResp.Usage.PromptTokensDetails != nil {
			usage.CacheReadInputTokens = openaiResp.Usage.PromptTokensDetails.CachedTokens
		}
	}

	return &model.AIEmbedResponse{
		Model:      openaiResp.Model,
		Embeddings: embeddings,
		Usage:      usage,
	}, nil
}

// buildChatRequest builds the OpenAI chat request body.
func (a *OpenAIAdapter) buildChatRequest(req *model.AIChatRequest, m *model.AIModel) map[string]any {
	body := map[string]any{
		"model":    m.ID,
		"messages": req.Messages,
	}

	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		body["top_p"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		body["stop"] = req.Stop
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
		if req.ToolChoice != nil {
			body["tool_choice"] = req.ToolChoice
		}
	}

	return body
}

// doRequest performs an HTTP request to the OpenAI API.
func (a *OpenAIAdapter) doRequest(ctx context.Context, p *model.AIProvider, apiKey, path string, body map[string]any) (io.ReadCloser, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return resp.Body, nil
}

// Compile-time interface assertions
var _ outbound.AIVendorAdapterPort = (*OpenAIAdapter)(nil)
