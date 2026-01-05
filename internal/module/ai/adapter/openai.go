package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/uniedit/server/internal/module/ai/provider"
)

// OpenAIAdapter implements the Adapter interface for OpenAI.
type OpenAIAdapter struct {
	*BaseAdapter
	client *http.Client
}

// NewOpenAIAdapter creates a new OpenAI adapter.
func NewOpenAIAdapter() *OpenAIAdapter {
	return &OpenAIAdapter{
		BaseAdapter: NewBaseAdapter(
			provider.CapabilityChat,
			provider.CapabilityStream,
			provider.CapabilityVision,
			provider.CapabilityTools,
			provider.CapabilityJSON,
			provider.CapabilityEmbedding,
		),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Type returns the adapter type.
func (a *OpenAIAdapter) Type() provider.ProviderType {
	return provider.ProviderTypeOpenAI
}

// Chat performs a non-streaming chat completion.
func (a *OpenAIAdapter) Chat(ctx context.Context, req *ChatRequest, model *provider.Model, prov *provider.Provider) (*ChatResponse, error) {
	// Build request body
	body := a.buildChatRequest(req, model)

	// Make API request
	respBody, err := a.doRequest(ctx, prov, "/chat/completions", body)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	// Parse response
	var openaiResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Index        int      `json:"index"`
			Message      *Message `json:"message"`
			FinishReason string   `json:"finish_reason"`
		} `json:"choices"`
		Usage *Usage `json:"usage"`
	}

	if err := json.NewDecoder(respBody).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:           openaiResp.ID,
		Model:        openaiResp.Model,
		Message:      openaiResp.Choices[0].Message,
		FinishReason: openaiResp.Choices[0].FinishReason,
		Usage:        openaiResp.Usage,
	}, nil
}

// ChatStream performs a streaming chat completion.
func (a *OpenAIAdapter) ChatStream(ctx context.Context, req *ChatRequest, model *provider.Model, prov *provider.Provider) (<-chan *ChatChunk, error) {
	// Build request body with streaming enabled
	body := a.buildChatRequest(req, model)
	body["stream"] = true

	// Make API request
	respBody, err := a.doRequest(ctx, prov, "/chat/completions", body)
	if err != nil {
		return nil, err
	}

	// Create channel for chunks
	chunks := make(chan *ChatChunk, 100)

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
func (a *OpenAIAdapter) Embed(ctx context.Context, input []string, model *provider.Model, prov *provider.Provider) (*EmbedResponse, error) {
	body := map[string]any{
		"model": model.ID,
		"input": input,
	}

	respBody, err := a.doRequest(ctx, prov, "/embeddings", body)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	var openaiResp struct {
		Model string `json:"model"`
		Data  []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Usage *Usage `json:"usage"`
	}

	if err := json.NewDecoder(respBody).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	embeddings := make([][]float64, len(openaiResp.Data))
	for _, d := range openaiResp.Data {
		embeddings[d.Index] = d.Embedding
	}

	return &EmbedResponse{
		Model:      openaiResp.Model,
		Embeddings: embeddings,
		Usage:      openaiResp.Usage,
	}, nil
}

// HealthCheck performs a health check.
func (a *OpenAIAdapter) HealthCheck(ctx context.Context, prov *provider.Provider) error {
	// Simple models list request to check connectivity
	req, err := http.NewRequestWithContext(ctx, "GET", prov.BaseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+prov.APIKey)

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

// buildChatRequest builds the OpenAI chat request body.
func (a *OpenAIAdapter) buildChatRequest(req *ChatRequest, model *provider.Model) map[string]any {
	body := map[string]any{
		"model":    model.ID,
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
func (a *OpenAIAdapter) doRequest(ctx context.Context, prov *provider.Provider, path string, body map[string]any) (io.ReadCloser, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", prov.BaseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+prov.APIKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}
