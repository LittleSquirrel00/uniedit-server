package adapter

import (
	"github.com/uniedit/server/internal/module/ai/provider"
)

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model       string         `json:"model"`
	Messages    []*Message     `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	TopP        *float64       `json:"top_p,omitempty"`
	Stop        []string       `json:"stop,omitempty"`
	Tools       []*Tool        `json:"tools,omitempty"`
	ToolChoice  any            `json:"tool_choice,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role       string      `json:"role"`
	Content    any         `json:"content"` // string or []ContentPart
	Name       string      `json:"name,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	ToolCalls  []*ToolCall `json:"tool_calls,omitempty"`
}

// ContentPart represents a multimodal content part.
type ContentPart struct {
	Type     string    `json:"type"` // text, image_url
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL reference.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // auto, low, high
}

// Tool represents a tool definition.
type Tool struct {
	Type     string    `json:"type"` // function
	Function *Function `json:"function"`
}

// Function represents a function definition.
type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolCall represents a tool call in a response.
type ToolCall struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"` // function
	Function *FunctionCall `json:"function"`
}

// FunctionCall represents a function call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID           string   `json:"id"`
	Model        string   `json:"model"`
	Message      *Message `json:"message"`
	FinishReason string   `json:"finish_reason"`
	Usage        *Usage   `json:"usage"`
}

// ChatChunk represents a streaming chat chunk.
type ChatChunk struct {
	ID           string `json:"id"`
	Model        string `json:"model"`
	Delta        *Delta `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Delta represents incremental content.
type Delta struct {
	Role      string      `json:"role,omitempty"`
	Content   string      `json:"content,omitempty"`
	ToolCalls []*ToolCall `json:"tool_calls,omitempty"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbedRequest represents an embedding request.
type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbedResponse represents an embedding response.
type EmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
	Usage      *Usage      `json:"usage,omitempty"`
}

// GetTextContent extracts text content from a message.
func (m *Message) GetTextContent() string {
	switch v := m.Content.(type) {
	case string:
		return v
	case []any:
		var text string
		for _, part := range v {
			if p, ok := part.(map[string]any); ok {
				if t, ok := p["type"].(string); ok && t == "text" {
					if txt, ok := p["text"].(string); ok {
						text += txt
					}
				}
			}
		}
		return text
	default:
		return ""
	}
}

// HasImages checks if the message contains images.
func (m *Message) HasImages() bool {
	switch v := m.Content.(type) {
	case []any:
		for _, part := range v {
			if p, ok := part.(map[string]any); ok {
				if t, ok := p["type"].(string); ok && t == "image_url" {
					return true
				}
			}
		}
	}
	return false
}

// RequiresCapability checks if the request requires a specific capability.
func (r *ChatRequest) RequiresCapability(cap provider.Capability) bool {
	switch cap {
	case provider.CapabilityStream:
		return r.Stream
	case provider.CapabilityTools:
		return len(r.Tools) > 0
	case provider.CapabilityVision:
		for _, msg := range r.Messages {
			if msg.HasImages() {
				return true
			}
		}
	}
	return false
}
