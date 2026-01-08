package ai

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	id           string
	model        string
	message      *Message
	finishReason string
	usage        *Usage
}

// NewChatResponse creates a new chat response.
func NewChatResponse(id, model string, message *Message) *ChatResponse {
	return &ChatResponse{
		id:      id,
		model:   model,
		message: message,
	}
}

// ReconstructChatResponse reconstructs a response from persistence.
func ReconstructChatResponse(
	id string,
	model string,
	message *Message,
	finishReason string,
	usage *Usage,
) *ChatResponse {
	return &ChatResponse{
		id:           id,
		model:        model,
		message:      message,
		finishReason: finishReason,
		usage:        usage,
	}
}

// Getters
func (r *ChatResponse) ID() string           { return r.id }
func (r *ChatResponse) Model() string        { return r.model }
func (r *ChatResponse) Message() *Message    { return r.message }
func (r *ChatResponse) FinishReason() string { return r.finishReason }
func (r *ChatResponse) Usage() *Usage        { return r.usage }

// Setters
func (r *ChatResponse) SetFinishReason(fr string) { r.finishReason = fr }
func (r *ChatResponse) SetUsage(u *Usage)         { r.usage = u }

// ChatChunk represents a streaming chat chunk.
type ChatChunk struct {
	id           string
	model        string
	delta        *Delta
	finishReason string
}

// NewChatChunk creates a new chat chunk.
func NewChatChunk(id, model string) *ChatChunk {
	return &ChatChunk{
		id:    id,
		model: model,
	}
}

// Getters
func (c *ChatChunk) ID() string           { return c.id }
func (c *ChatChunk) Model() string        { return c.model }
func (c *ChatChunk) Delta() *Delta        { return c.delta }
func (c *ChatChunk) FinishReason() string { return c.finishReason }

// Setters
func (c *ChatChunk) SetDelta(d *Delta)          { c.delta = d }
func (c *ChatChunk) SetFinishReason(fr string)  { c.finishReason = fr }

// Delta represents incremental content in streaming.
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

// NewUsage creates a new usage record.
func NewUsage(prompt, completion int) *Usage {
	return &Usage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      prompt + completion,
	}
}

// EmbedResponse represents an embedding response.
type EmbedResponse struct {
	model      string
	embeddings [][]float64
	usage      *Usage
}

// NewEmbedResponse creates a new embedding response.
func NewEmbedResponse(model string, embeddings [][]float64) *EmbedResponse {
	return &EmbedResponse{
		model:      model,
		embeddings: embeddings,
	}
}

// Getters
func (r *EmbedResponse) Model() string          { return r.model }
func (r *EmbedResponse) Embeddings() [][]float64 { return r.embeddings }
func (r *EmbedResponse) Usage() *Usage          { return r.usage }

// Setters
func (r *EmbedResponse) SetUsage(u *Usage) { r.usage = u }
