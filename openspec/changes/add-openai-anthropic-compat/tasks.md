# Tasks: Add OpenAI and Anthropic SDK Compatible APIs

## Phase 1: Type Definitions and Core Infrastructure

- [ ] **1.1** Create `handler/compat_types.go` with OpenAI-format request/response types
  - OpenAIChatRequest, OpenAIChatResponse, OpenAIChoice, OpenAIMessage
  - OpenAIStreamChoice, OpenAIStreamChunk
  - OpenAIEmbeddingRequest, OpenAIEmbeddingResponse
  - OpenAIModelsResponse, OpenAIModel
  - OpenAIErrorResponse

- [ ] **1.2** Create `handler/anthropic_types.go` with Anthropic-format types
  - AnthropicRequest, AnthropicResponse
  - AnthropicContent, AnthropicContentBlock
  - AnthropicUsage, AnthropicStreamEvent
  - AnthropicErrorResponse

## Phase 2: OpenAI Compatible Handler

- [ ] **2.1** Create `handler/openai_compat.go` with OpenAICompatHandler struct
  - Constructor with llmService and registry dependencies
  - RegisterRoutes method for route mounting

- [ ] **2.2** Implement ChatCompletions endpoint
  - Parse OpenAI-format request
  - Transform to internal format
  - Call llmService.Chat()
  - Transform response to OpenAI format
  - Handle stream=true for streaming

- [ ] **2.3** Implement Embeddings endpoint
  - Parse OpenAI-format request
  - Call llmService.Embed()
  - Transform response to OpenAI format

- [ ] **2.4** Implement ListModels endpoint
  - Fetch models from registry
  - Transform to OpenAI models list format

- [ ] **2.5** Implement streaming for ChatCompletions
  - Detect stream=true in request
  - Transform chunks to OpenAI streaming format
  - Send proper SSE events with data: prefix
  - Send [DONE] marker at end

## Phase 3: Anthropic Compatible Handler

- [ ] **3.1** Create `handler/anthropic_compat.go` with AnthropicCompatHandler struct
  - Constructor with llmService dependency
  - RegisterRoutes method for route mounting

- [ ] **3.2** Implement Messages endpoint (non-streaming)
  - Parse Anthropic-format request
  - Transform to internal format (handle content array format)
  - Call llmService.Chat()
  - Transform response to Anthropic format

- [ ] **3.3** Implement Messages streaming
  - Detect stream=true in request
  - Send Anthropic-style typed events:
    - message_start
    - content_block_start
    - content_block_delta
    - content_block_stop
    - message_delta (for stop_reason)
    - message_stop

## Phase 4: Error Handling

- [ ] **4.1** Create error translation utilities
  - `toOpenAIError()` - transform to OpenAI error format
  - `toAnthropicError()` - transform to Anthropic error format
  - Map internal error types to appropriate status codes

## Phase 5: Integration

- [ ] **5.1** Update `handler/routes.go`
  - Add OpenAICompat and AnthropicCompat to Handlers struct
  - Update NewHandlers to initialize compat handlers
  - Update RegisterRoutes to mount compat routes

- [ ] **5.2** Update `module.go`
  - Pass required dependencies to compat handlers

- [ ] **5.3** Update route mounting in `app/app.go`
  - Mount `/v1/*` routes at root level (not under /api)
  - Apply auth middleware to compat routes

## Phase 6: Testing and Documentation

- [ ] **6.1** Add unit tests for format translation
  - Test OpenAI request/response transformation
  - Test Anthropic request/response transformation
  - Test streaming chunk transformation

- [ ] **6.2** Add integration tests
  - Test end-to-end with mock service
  - Test streaming behavior

- [ ] **6.3** Update Swagger annotations
  - Add annotations to compat handlers
  - Generate updated API documentation

- [ ] **6.4** Update README with usage examples
  - OpenAI Python SDK configuration
  - Anthropic SDK configuration
  - curl examples

## Dependencies

```
1.1, 1.2 → 2.1, 3.1 (types needed first)
2.1 → 2.2 → 2.5 (sequential OpenAI)
2.1 → 2.3, 2.4 (parallel)
3.1 → 3.2 → 3.3 (sequential Anthropic)
4.1 → 2.2, 3.2 (errors needed for handlers)
5.1 → 5.2 → 5.3 (integration sequence)
6.* can run after Phase 5
```

## Parallelizable Work

- Phase 2 (OpenAI) and Phase 3 (Anthropic) can run in parallel after Phase 1
- Tasks 2.3, 2.4 can run in parallel after 2.1
- All Phase 6 tasks can run in parallel after Phase 5
