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

const (
	anthropicAPIVersion = "2023-06-01"
	anthropicBeta       = "messages-2024-12-19"
)

// AnthropicAdapter implements the AIVendorAdapterPort interface for Anthropic.
type AnthropicAdapter struct {
	*BaseAdapter
	client *http.Client
}

// NewAnthropicAdapter creates a new Anthropic adapter with the given HTTP client.
func NewAnthropicAdapter(client *http.Client) *AnthropicAdapter {
	return &AnthropicAdapter{
		BaseAdapter: NewBaseAdapter(
			model.AICapabilityChat,
			model.AICapabilityStream,
			model.AICapabilityVision,
			model.AICapabilityTools,
		),
		client: client,
	}
}

// Type returns the adapter type.
func (a *AnthropicAdapter) Type() model.AIProviderType {
	return model.AIProviderTypeAnthropic
}

// HealthCheck performs a health check on the provider.
func (a *AnthropicAdapter) HealthCheck(ctx context.Context, provider *model.AIProvider, apiKey string) error {
	// Simple request to check connectivity
	body := map[string]any{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 1,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}

	respBody, err := a.doRequest(ctx, provider, apiKey, "/messages", body)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	respBody.Close()

	return nil
}

// Chat performs a non-streaming chat completion.
func (a *AnthropicAdapter) Chat(ctx context.Context, req *model.AIChatRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIChatResponse, error) {
	body := a.buildRequest(req, m)

	respBody, err := a.doRequest(ctx, p, apiKey, "/messages", body)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	var anthropicResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(respBody).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Extract text content
	var content string
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &model.AIChatResponse{
		ID:    anthropicResp.ID,
		Model: anthropicResp.Model,
		Message: &model.AIChatMessage{
			Role:    anthropicResp.Role,
			Content: content,
		},
		FinishReason: a.mapStopReason(anthropicResp.StopReason),
		Usage: &model.AIUsage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}, nil
}

// ChatStream performs a streaming chat completion.
func (a *AnthropicAdapter) ChatStream(ctx context.Context, req *model.AIChatRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (<-chan *model.AIChatChunk, error) {
	body := a.buildRequest(req, m)
	body["stream"] = true

	respBody, err := a.doRequest(ctx, p, apiKey, "/messages", body)
	if err != nil {
		return nil, err
	}

	chunks := make(chan *model.AIChatChunk, 100)

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
				return
			}

			chunk, err := ParseAnthropicEvent(event.Event, event.Data)
			if err != nil {
				if err == io.EOF {
					return
				}
				continue
			}

			if chunk == nil {
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

// Embed is not supported by Anthropic.
func (a *AnthropicAdapter) Embed(ctx context.Context, req *model.AIEmbedRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIEmbedResponse, error) {
	return nil, fmt.Errorf("embedding not supported by Anthropic")
}

// buildRequest builds the Anthropic request body.
func (a *AnthropicAdapter) buildRequest(req *model.AIChatRequest, m *model.AIModel) map[string]any {
	// Extract system message and convert messages
	var system string
	messages := make([]map[string]any, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if s, ok := msg.Content.(string); ok {
				system = s
			}
			continue
		}

		mm := map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		}
		messages = append(messages, mm)
	}

	body := map[string]any{
		"model":    m.ID,
		"messages": messages,
	}

	if system != "" {
		body["system"] = system
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = m.MaxOutputTokens
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	body["max_tokens"] = maxTokens

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		body["top_p"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		body["stop_sequences"] = req.Stop
	}
	if len(req.Tools) > 0 {
		body["tools"] = a.convertTools(req.Tools)
	}

	return body
}

// convertTools converts OpenAI tool format to Anthropic format.
func (a *AnthropicAdapter) convertTools(tools []*model.AITool) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, t := range tools {
		result[i] = map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": t.Function.Parameters,
		}
	}
	return result
}

// mapStopReason maps Anthropic stop reason to OpenAI format.
func (a *AnthropicAdapter) mapStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return reason
	}
}

// doRequest performs an HTTP request to the Anthropic API.
func (a *AnthropicAdapter) doRequest(ctx context.Context, p *model.AIProvider, apiKey, path string, body map[string]any) (io.ReadCloser, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("anthropic-beta", anthropicBeta)

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
var _ outbound.AIVendorAdapterPort = (*AnthropicAdapter)(nil)
