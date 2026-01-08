package ai

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	model       string
	messages    []*Message
	maxTokens   int
	temperature *float64
	topP        *float64
	stop        []string
	tools       []*Tool
	toolChoice  any
	stream      bool
	metadata    map[string]any
}

// NewChatRequest creates a new chat request.
func NewChatRequest(model string, messages []*Message) *ChatRequest {
	return &ChatRequest{
		model:    model,
		messages: messages,
		metadata: make(map[string]any),
	}
}

// Getters
func (r *ChatRequest) Model() string            { return r.model }
func (r *ChatRequest) Messages() []*Message     { return r.messages }
func (r *ChatRequest) MaxTokens() int           { return r.maxTokens }
func (r *ChatRequest) Temperature() *float64    { return r.temperature }
func (r *ChatRequest) TopP() *float64           { return r.topP }
func (r *ChatRequest) Stop() []string           { return r.stop }
func (r *ChatRequest) Tools() []*Tool           { return r.tools }
func (r *ChatRequest) ToolChoice() any          { return r.toolChoice }
func (r *ChatRequest) Stream() bool             { return r.stream }
func (r *ChatRequest) Metadata() map[string]any { return r.metadata }

// Setters
func (r *ChatRequest) SetMaxTokens(n int)              { r.maxTokens = n }
func (r *ChatRequest) SetTemperature(t float64)        { r.temperature = &t }
func (r *ChatRequest) SetTopP(p float64)               { r.topP = &p }
func (r *ChatRequest) SetStop(s []string)              { r.stop = s }
func (r *ChatRequest) SetTools(t []*Tool)              { r.tools = t }
func (r *ChatRequest) SetToolChoice(tc any)            { r.toolChoice = tc }
func (r *ChatRequest) SetStream(s bool)                { r.stream = s }
func (r *ChatRequest) SetMetadata(m map[string]any)    { r.metadata = m }
func (r *ChatRequest) AddMetadata(key string, val any) { r.metadata[key] = val }

// RequiresCapability checks if the request requires a specific capability.
func (r *ChatRequest) RequiresCapability(cap Capability) bool {
	switch cap {
	case CapabilityStream:
		return r.stream
	case CapabilityTools:
		return len(r.tools) > 0
	case CapabilityVision:
		for _, msg := range r.messages {
			if msg.HasImages() {
				return true
			}
		}
	}
	return false
}

// Message represents a chat message.
type Message struct {
	role       string
	content    any // string or []ContentPart
	name       string
	toolCallID string
	toolCalls  []*ToolCall
}

// NewTextMessage creates a new text message.
func NewTextMessage(role, content string) *Message {
	return &Message{
		role:    role,
		content: content,
	}
}

// NewMultimodalMessage creates a new multimodal message.
func NewMultimodalMessage(role string, parts []ContentPart) *Message {
	return &Message{
		role:    role,
		content: parts,
	}
}

// ReconstructMessage reconstructs a message from persistence.
func ReconstructMessage(
	role string,
	content any,
	name string,
	toolCallID string,
	toolCalls []*ToolCall,
) *Message {
	return &Message{
		role:       role,
		content:    content,
		name:       name,
		toolCallID: toolCallID,
		toolCalls:  toolCalls,
	}
}

// Getters
func (m *Message) Role() string          { return m.role }
func (m *Message) Content() any          { return m.content }
func (m *Message) Name() string          { return m.name }
func (m *Message) ToolCallID() string    { return m.toolCallID }
func (m *Message) ToolCalls() []*ToolCall { return m.toolCalls }

// Setters
func (m *Message) SetName(n string)             { m.name = n }
func (m *Message) SetToolCallID(id string)      { m.toolCallID = id }
func (m *Message) SetToolCalls(tc []*ToolCall)  { m.toolCalls = tc }

// GetTextContent extracts text content from a message.
func (m *Message) GetTextContent() string {
	switch v := m.content.(type) {
	case string:
		return v
	case []ContentPart:
		var text string
		for _, part := range v {
			if part.Type == "text" {
				text += part.Text
			}
		}
		return text
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
	switch v := m.content.(type) {
	case []ContentPart:
		for _, part := range v {
			if part.Type == "image_url" {
				return true
			}
		}
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

// EmbedRequest represents an embedding request.
type EmbedRequest struct {
	model string
	input []string
}

// NewEmbedRequest creates a new embedding request.
func NewEmbedRequest(model string, input []string) *EmbedRequest {
	return &EmbedRequest{
		model: model,
		input: input,
	}
}

// Getters
func (r *EmbedRequest) Model() string   { return r.model }
func (r *EmbedRequest) Input() []string { return r.input }
