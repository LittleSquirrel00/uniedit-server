# AI LLM Service Capability

## ADDED Requirements

### Requirement: Chat API

The system SHALL provide chat completion API with intelligent routing.

#### Scenario: Non-streaming chat

- **WHEN** user sends POST /api/v1/ai/chat with messages
- **THEN** route to best available model
- **AND** return ChatResponse with message, usage, and routing info

#### Scenario: Streaming chat

- **WHEN** user sends POST /api/v1/ai/chat/stream with Accept: text/event-stream
- **THEN** route to streaming-capable model
- **AND** stream SSE events with delta content

#### Scenario: Model specification

- **WHEN** request specifies model ID (e.g., "gpt-4o")
- **THEN** use specified model if available
- **AND** return error if model not found or disabled

#### Scenario: Auto model selection

- **WHEN** request specifies model as "auto"
- **THEN** use routing manager to select optimal model

#### Scenario: Vision request

- **WHEN** messages contain image_url content
- **THEN** route to vision-capable model
- **AND** transform content to provider format

#### Scenario: Tool calling

- **WHEN** request includes tools definition
- **THEN** route to tools-capable model
- **AND** include tool_calls in response if model invokes tools

### Requirement: Embedding API

The system SHALL provide text embedding generation with caching.

#### Scenario: Generate embeddings

- **WHEN** user requests embeddings for text input
- **THEN** check embedding cache first
- **AND** call provider API for cache misses
- **AND** cache results for future requests

#### Scenario: Batch embedding

- **WHEN** user requests embeddings for multiple texts
- **THEN** return cached results for hits
- **AND** batch API call for misses

### Requirement: Routing Info

The system SHALL include routing metadata in responses.

#### Scenario: Include routing info

- **WHEN** response is generated
- **THEN** include _routing object with provider_used, model_used, latency_ms
- **AND** optionally include cost_usd if billing enabled

### Requirement: Error Handling

The system SHALL handle provider errors gracefully.

#### Scenario: Rate limit error

- **WHEN** provider returns rate limit error
- **THEN** if fallback enabled, retry with next model
- **AND** if no fallback, return 429 with retry-after

#### Scenario: Provider timeout

- **WHEN** provider request times out
- **THEN** if fallback enabled, retry with next model
- **AND** mark provider health as degraded

#### Scenario: Invalid request

- **WHEN** request validation fails
- **THEN** return 400 with specific error message

### Requirement: Admin API

The system SHALL provide admin endpoints for configuration management.

#### Scenario: List providers

- **WHEN** admin requests GET /api/v1/admin/ai/providers
- **THEN** return all providers with model counts

#### Scenario: Create provider

- **WHEN** admin sends POST /api/v1/admin/ai/providers
- **THEN** create provider with encrypted api_key
- **AND** refresh provider registry

#### Scenario: List models

- **WHEN** admin requests GET /api/v1/admin/ai/models
- **THEN** return all models with provider info

#### Scenario: Create model

- **WHEN** admin sends POST /api/v1/admin/ai/models
- **THEN** create model and associate with provider

#### Scenario: Manage groups

- **WHEN** admin creates/updates/deletes groups
- **THEN** persist changes and refresh group manager
