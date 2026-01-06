# ai-sdk-compat Specification

## Purpose

Provide SDK-compatible API endpoints that allow users to use official OpenAI and Anthropic client libraries with UniEdit server by simply changing the base URL.

## ADDED Requirements

### Requirement: OpenAI Chat Completions API

The system SHALL provide an OpenAI-compatible chat completions endpoint.

#### Scenario: Non-streaming chat completion

- **WHEN** client sends POST /v1/chat/completions with OpenAI-format request
- **THEN** transform request to internal format
- **AND** route and execute via LLM service
- **AND** return response in OpenAI format with `choices`, `usage`, `id`

#### Scenario: Streaming chat completion

- **WHEN** client sends POST /v1/chat/completions with `stream: true`
- **THEN** set Content-Type to text/event-stream
- **AND** stream SSE events with `data:` prefix containing JSON chunks
- **AND** include `delta` object with incremental content
- **AND** send `data: [DONE]` at stream end

#### Scenario: Vision request via OpenAI format

- **WHEN** request contains messages with `image_url` content type
- **THEN** transform to internal vision format
- **AND** route to vision-capable model

#### Scenario: Tool calling via OpenAI format

- **WHEN** request includes `tools` array with function definitions
- **THEN** transform to internal tools format
- **AND** include `tool_calls` in response if model invokes tools

### Requirement: OpenAI Embeddings API

The system SHALL provide an OpenAI-compatible embeddings endpoint.

#### Scenario: Generate embeddings

- **WHEN** client sends POST /v1/embeddings with `model` and `input`
- **THEN** call embedding service
- **AND** return response with `data` array of embedding objects
- **AND** include `usage` with token counts

#### Scenario: Batch embeddings

- **WHEN** input is an array of strings
- **THEN** return embeddings for each input with correct indices

### Requirement: OpenAI Models API

The system SHALL provide an OpenAI-compatible models listing endpoint.

#### Scenario: List available models

- **WHEN** client sends GET /v1/models
- **THEN** return list of available models
- **AND** format as OpenAI models response with `id`, `object`, `created`, `owned_by`

### Requirement: Anthropic Messages API

The system SHALL provide an Anthropic-compatible messages endpoint.

#### Scenario: Non-streaming message

- **WHEN** client sends POST /v1/messages with Anthropic-format request
- **THEN** transform content array to internal format
- **AND** route and execute via LLM service
- **AND** return response with `content` array, `usage`, `stop_reason`

#### Scenario: Streaming message

- **WHEN** client sends POST /v1/messages with `stream: true`
- **THEN** set Content-Type to text/event-stream
- **AND** send typed SSE events:
  - `event: message_start` with message metadata
  - `event: content_block_start` with content block metadata
  - `event: content_block_delta` with incremental text
  - `event: content_block_stop` when block ends
  - `event: message_delta` with stop_reason
  - `event: message_stop` at stream end

#### Scenario: System prompt via Anthropic format

- **WHEN** request includes `system` field (string)
- **THEN** transform to internal system message format

### Requirement: SDK Error Responses

The system SHALL return errors in SDK-expected formats.

#### Scenario: OpenAI error format

- **WHEN** error occurs on OpenAI-compatible endpoint
- **THEN** return error in OpenAI format:
  ```json
  {"error": {"message": "...", "type": "...", "code": "..."}}
  ```
- **AND** use appropriate HTTP status codes

#### Scenario: Anthropic error format

- **WHEN** error occurs on Anthropic-compatible endpoint
- **THEN** return error in Anthropic format:
  ```json
  {"type": "error", "error": {"type": "...", "message": "..."}}
  ```
- **AND** use appropriate HTTP status codes

#### Scenario: Rate limit errors

- **WHEN** user exceeds rate limits
- **THEN** return 429 with appropriate error format
- **AND** include retry-after header if available

### Requirement: Authentication Compatibility

The system SHALL support SDK authentication patterns.

#### Scenario: Bearer token authentication

- **WHEN** request includes `Authorization: Bearer <token>`
- **THEN** validate as UniEdit API key
- **AND** associate request with user for billing

#### Scenario: Anthropic API key header

- **WHEN** request includes `x-api-key: <token>`
- **THEN** validate as UniEdit API key (alternative to Bearer)
