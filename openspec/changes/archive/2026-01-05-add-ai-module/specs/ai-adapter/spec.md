# AI Adapter Capability

## ADDED Requirements

### Requirement: Adapter Interface

The system SHALL provide a unified Adapter interface for all LLM providers.

#### Scenario: Chat completion

- **WHEN** system calls Adapter.Chat with request, model, and provider
- **THEN** return ChatResponse with message, usage, and finish_reason
- **AND** handle provider-specific request/response transformation

#### Scenario: Streaming chat

- **WHEN** system calls Adapter.ChatStream with request
- **THEN** return a channel of ChatChunk
- **AND** parse SSE stream according to provider format

#### Scenario: Embedding generation

- **WHEN** system calls Adapter.Embed with text input
- **THEN** return vector embeddings as [][]float64
- **AND** handle batch requests efficiently

#### Scenario: Health check

- **WHEN** system calls Adapter.HealthCheck
- **THEN** perform lightweight API call to verify connectivity
- **AND** return error if provider is unreachable

### Requirement: OpenAI Adapter

The system SHALL provide an OpenAI adapter using the official SDK.

#### Scenario: OpenAI chat request

- **WHEN** adapter receives chat request for OpenAI provider
- **THEN** transform to OpenAI API format
- **AND** handle vision content (image_url) if present

#### Scenario: OpenAI streaming

- **WHEN** adapter receives streaming request
- **THEN** parse OpenAI SSE format (data: {...})
- **AND** yield ChatChunk for each delta

#### Scenario: OpenAI tool calls

- **WHEN** request includes tools definition
- **THEN** include tools in API request
- **AND** parse tool_calls in response

### Requirement: Anthropic Adapter

The system SHALL provide an Anthropic adapter for Claude models.

#### Scenario: Anthropic chat request

- **WHEN** adapter receives chat request for Anthropic provider
- **THEN** transform messages to Anthropic format
- **AND** handle system message separately (Anthropic uses system parameter)

#### Scenario: Anthropic streaming

- **WHEN** adapter receives streaming request
- **THEN** parse Anthropic SSE format (event types: message_start, content_block_delta, etc.)
- **AND** yield ChatChunk for each content delta

#### Scenario: Anthropic vision

- **WHEN** request includes image content
- **THEN** transform to Anthropic image format (base64 or URL)

### Requirement: Generic Adapter

The system SHALL provide a generic adapter for OpenAI-compatible APIs.

#### Scenario: Generic provider request

- **WHEN** adapter receives request for generic provider type
- **THEN** use OpenAI API format
- **AND** send to provider's base_url

#### Scenario: DeepSeek compatibility

- **WHEN** provider is configured as generic with DeepSeek base_url
- **THEN** requests work without modification

### Requirement: Adapter Registry

The system SHALL provide an AdapterRegistry to manage adapter instances.

#### Scenario: Get adapter by type

- **WHEN** system requests adapter for provider type
- **THEN** return singleton adapter instance for that type

#### Scenario: Register custom adapter

- **WHEN** system registers a new adapter type
- **THEN** the adapter is available for providers of that type
