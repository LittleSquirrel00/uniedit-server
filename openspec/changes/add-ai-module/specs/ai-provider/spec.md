# AI Provider Capability

## ADDED Requirements

### Requirement: Provider Management

The system SHALL manage AI providers with database persistence and in-memory caching for high-performance access.

#### Scenario: Create provider

- **WHEN** admin creates a provider with name, type, base_url, and api_key
- **THEN** the provider is stored in database with encrypted api_key
- **AND** the provider is added to in-memory cache

#### Scenario: List providers

- **WHEN** admin requests provider list
- **THEN** return all providers with model counts
- **AND** api_key is never exposed in response

#### Scenario: Update provider

- **WHEN** admin updates provider configuration
- **THEN** the database is updated
- **AND** the in-memory cache is refreshed

#### Scenario: Delete provider

- **WHEN** admin deletes a provider
- **THEN** the provider and its models are removed
- **AND** the cache is invalidated

### Requirement: Model Management

The system SHALL manage AI models with capabilities, pricing, and context window information.

#### Scenario: Create model

- **WHEN** admin creates a model with provider_id, capabilities, context_window, and pricing
- **THEN** the model is stored in database
- **AND** the model is associated with its provider in cache

#### Scenario: Query models by capability

- **WHEN** system queries models with required capabilities
- **THEN** return only models that have all required capabilities
- **AND** filter out disabled models

#### Scenario: Model cost calculation

- **WHEN** system calculates cost for a model
- **THEN** return input_cost_per_1k and output_cost_per_1k
- **AND** provide average_cost_per_1k helper method

### Requirement: Provider Registry

The system SHALL provide a ProviderRegistry for efficient in-memory access to providers and models.

#### Scenario: Registry initialization

- **WHEN** server starts
- **THEN** load all enabled providers and models from database
- **AND** build in-memory index by type and capability

#### Scenario: Get provider by ID

- **WHEN** system requests provider by ID
- **THEN** return provider from memory cache in O(1) time

#### Scenario: Get models by capability

- **WHEN** system requests models with specific capabilities
- **THEN** return filtered models from capability index

#### Scenario: Registry refresh

- **WHEN** admin triggers cache refresh or refresh interval elapses
- **THEN** reload all data from database
- **AND** update indexes atomically

### Requirement: Health Monitoring

The system SHALL monitor provider health status with circuit breaker pattern.

#### Scenario: Health check success

- **WHEN** health check request succeeds
- **THEN** mark provider as healthy
- **AND** reset failure counter

#### Scenario: Health check failure

- **WHEN** health check request fails
- **THEN** increment failure counter
- **AND** if failures exceed threshold, open circuit breaker

#### Scenario: Circuit breaker open

- **WHEN** circuit breaker is open
- **THEN** skip provider in routing decisions
- **AND** attempt recovery after cooldown period

#### Scenario: Circuit breaker recovery

- **WHEN** cooldown period elapses
- **THEN** allow single test request (half-open state)
- **AND** close circuit if request succeeds
