## MODIFIED Requirements

### Requirement: Adapter Interface

The system SHALL provide segregated interfaces for LLM adapter capabilities following Interface Segregation Principle.

#### Scenario: Text completion adapter

- **WHEN** system needs chat completion functionality
- **THEN** use `TextAdapter` interface with Chat and ChatStream methods
- **AND** implementations provide provider-specific request/response transformation

#### Scenario: Embedding adapter

- **WHEN** system needs embedding generation functionality
- **THEN** use `EmbeddingAdapter` interface with Embed method
- **AND** handle batch requests efficiently

#### Scenario: Health check adapter

- **WHEN** system needs to check provider health
- **THEN** use `HealthChecker` interface with HealthCheck method
- **AND** perform lightweight API call to verify connectivity

#### Scenario: Capability query

- **WHEN** system needs to query adapter capabilities
- **THEN** use `CapabilityProvider` interface with Type and SupportsCapability methods

#### Scenario: Composite adapter

- **WHEN** system needs full adapter functionality
- **THEN** use composite `Adapter` interface embedding all sub-interfaces
- **AND** maintain backwards compatibility with existing code

### Requirement: OpenAI Adapter

The system SHALL provide an OpenAI adapter implementing all segregated interfaces.

#### Scenario: OpenAI chat request

- **WHEN** adapter receives chat request for OpenAI provider
- **THEN** transform to OpenAI API format via TextAdapter interface
- **AND** handle vision content (image_url) if present

#### Scenario: OpenAI streaming

- **WHEN** adapter receives streaming request
- **THEN** parse OpenAI SSE format (data: {...}) via TextAdapter.ChatStream
- **AND** yield ChatChunk for each delta

#### Scenario: OpenAI tool calls

- **WHEN** request includes tools definition
- **THEN** include tools in API request
- **AND** parse tool_calls in response

### Requirement: Anthropic Adapter

The system SHALL provide an Anthropic adapter implementing all segregated interfaces.

#### Scenario: Anthropic chat request

- **WHEN** adapter receives chat request for Anthropic provider
- **THEN** transform messages to Anthropic format via TextAdapter
- **AND** handle system message separately (Anthropic uses system parameter)

#### Scenario: Anthropic streaming

- **WHEN** adapter receives streaming request
- **THEN** parse Anthropic SSE format via TextAdapter.ChatStream
- **AND** yield ChatChunk for each content delta

#### Scenario: Anthropic vision

- **WHEN** request includes image content
- **THEN** transform to Anthropic image format (base64 or URL)

### Requirement: Generic Adapter

The system SHALL provide a generic adapter for OpenAI-compatible APIs implementing all segregated interfaces.

#### Scenario: Generic provider request

- **WHEN** adapter receives request for generic provider type
- **THEN** use OpenAI API format via TextAdapter
- **AND** send to provider's base_url

#### Scenario: DeepSeek compatibility

- **WHEN** provider is configured as generic with DeepSeek base_url
- **THEN** requests work without modification

### Requirement: Adapter Registry

The system SHALL provide an AdapterRegistry managing adapters that implement segregated interfaces.

#### Scenario: Get adapter by type

- **WHEN** system requests adapter for provider type
- **THEN** return singleton adapter instance implementing requested interface

#### Scenario: Register custom adapter

- **WHEN** system registers a new adapter type
- **THEN** adapter is available for providers of that type
- **AND** adapter must implement at least one segregated interface
