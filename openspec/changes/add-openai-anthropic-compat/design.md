# Design: OpenAI and Anthropic SDK Compatible APIs

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                       API Layer                                  │
├────────────────┬────────────────────┬──────────────────────────┤
│  /api/v1/ai/*  │  /v1/chat/...      │  /v1/messages            │
│  (UniEdit)     │  (OpenAI Compat)   │  (Anthropic Compat)      │
├────────────────┴────────────────────┴──────────────────────────┤
│                    Format Translation                           │
│  ┌──────────────────┐         ┌────────────────────┐           │
│  │ OpenAI Adapter   │         │ Anthropic Adapter  │           │
│  │ - toInternal()   │         │ - toInternal()     │           │
│  │ - toOpenAI()     │         │ - toAnthropic()    │           │
│  └──────────────────┘         └────────────────────┘           │
├─────────────────────────────────────────────────────────────────┤
│                    LLM Service (Existing)                       │
│  Chat() / ChatStream() / Embed()                                │
├─────────────────────────────────────────────────────────────────┤
│                    Adapter Layer (Existing)                     │
│  OpenAI / Anthropic / Generic Adapters                          │
└─────────────────────────────────────────────────────────────────┘
```

## Design Decisions

### D1: Separate Handler Files

Create dedicated handler files for each SDK format:
- `openai_compat.go` - OpenAI-compatible handlers
- `anthropic_compat.go` - Anthropic-compatible handlers
- `compat_types.go` - Shared type definitions

**Rationale**: Clear separation of concerns, easier to maintain and extend.

### D2: Format Adapters in Handler Layer

Translation logic lives in handler layer, not adapter layer.

**Rationale**:
- Adapters handle provider-specific protocols (calling external APIs)
- Handlers handle client-facing protocols (serving requests)
- Different concerns, different layers

### D3: Route Mounting Strategy

Mount SDK-compatible routes at root level `/v1/*`:
- OpenAI: `/v1/chat/completions`, `/v1/embeddings`, `/v1/models`
- Anthropic: `/v1/messages`

**Rationale**:
- Matches official API paths for SDK compatibility
- Clear separation from custom `/api/v1/ai/*` routes

### D4: Streaming Implementation

OpenAI streaming uses SSE with `data:` prefix and JSON chunks.
Anthropic streaming uses typed SSE events with distinct event types.

```
OpenAI:
data: {"id":"chatcmpl-...","choices":[{"delta":{"content":"Hello"}}]}
data: [DONE]

Anthropic:
event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: message_stop
data: {"type":"message_stop"}
```

**Rationale**: Match each SDK's expected streaming format exactly.

### D5: Error Response Formats

Transform internal errors to SDK-specific formats:

```json
// OpenAI
{"error": {"message": "...", "type": "invalid_request_error", "code": "invalid_api_key"}}

// Anthropic
{"type": "error", "error": {"type": "authentication_error", "message": "..."}}
```

**Rationale**: SDK clients expect specific error structures.

## Component Design

### OpenAICompatHandler

```go
type OpenAICompatHandler struct {
    llmService *llm.Service
    registry   *provider.Registry
}

func (h *OpenAICompatHandler) RegisterRoutes(r *gin.RouterGroup) {
    r.POST("/v1/chat/completions", h.ChatCompletions)
    r.POST("/v1/embeddings", h.Embeddings)
    r.GET("/v1/models", h.ListModels)
}
```

### AnthropicCompatHandler

```go
type AnthropicCompatHandler struct {
    llmService *llm.Service
}

func (h *AnthropicCompatHandler) RegisterRoutes(r *gin.RouterGroup) {
    r.POST("/v1/messages", h.Messages)
}
```

### Type Definitions

Key OpenAI types:
- `OpenAIChatRequest` - Input format with `messages`, `model`, `stream`
- `OpenAIChatResponse` - Output with `choices`, `usage`, `id`
- `OpenAIStreamChunk` - Streaming chunk with `delta`

Key Anthropic types:
- `AnthropicRequest` - Input with `messages`, `model`, `max_tokens`
- `AnthropicResponse` - Output with `content`, `usage`, `stop_reason`
- `AnthropicStreamEvent` - Typed event (`message_start`, `content_block_delta`, etc.)

## Integration Points

### Authentication

Uses existing auth middleware with API key validation:
```
Authorization: Bearer sk-xxx    # UniEdit API key
x-api-key: sk-xxx               # Anthropic style (optional)
```

### Model Mapping

- Direct model IDs: `gpt-4o`, `claude-opus-4-5-20251101` → use as-is
- Model aliases: `gpt-4` → map to `gpt-4-turbo` (configurable)
- Auto routing: `auto` → use routing manager

### Billing Integration

Uses existing billing recording via userID from auth context.

## Testing Strategy

1. **Unit Tests**: Format translation functions
2. **Integration Tests**: End-to-end with mock LLM service
3. **SDK Tests**: Verify official SDK clients work correctly
