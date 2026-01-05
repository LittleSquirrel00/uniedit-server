# ai-routing Specification

## Purpose
TBD - created by archiving change add-ai-module. Update Purpose after archive.
## Requirements
### Requirement: Routing Manager

The system SHALL provide intelligent model routing based on configurable strategies.

#### Scenario: Route request to best model

- **WHEN** system calls RoutingManager.Route with context
- **THEN** execute strategy chain on candidate models
- **AND** return highest-scored model with routing reason

#### Scenario: No available candidates

- **WHEN** all candidates are filtered out by strategies
- **THEN** return error indicating which strategy eliminated candidates
- **AND** include available alternatives if any

#### Scenario: Auto model selection

- **WHEN** request specifies model as "auto"
- **THEN** use default group for task type
- **AND** apply full routing strategy chain

### Requirement: Strategy Chain

The system SHALL execute routing strategies in priority order.

#### Scenario: Strategy chain execution

- **WHEN** routing manager executes strategy chain
- **THEN** strategies run in descending priority order
- **AND** each strategy can filter and score candidates

#### Scenario: UserPreference strategy (priority 100)

- **WHEN** context includes preferred or excluded providers
- **THEN** filter out excluded providers
- **AND** add bonus score (+20/+30) to preferred models

#### Scenario: HealthFilter strategy (priority 90)

- **WHEN** strategy evaluates candidates
- **THEN** filter out providers with open circuit breakers
- **AND** filter out providers marked unhealthy

#### Scenario: CapabilityFilter strategy (priority 80)

- **WHEN** context requires specific capabilities (vision, tools, stream)
- **THEN** filter out models lacking required capabilities

#### Scenario: ContextWindow strategy (priority 70)

- **WHEN** context specifies minimum context window
- **THEN** filter out models with insufficient context window
- **AND** add bonus score for larger context windows

#### Scenario: CostOptimization strategy (priority 50)

- **WHEN** context specifies cost optimization
- **THEN** filter out models exceeding max_cost_per_1k
- **AND** score models inversely by cost

#### Scenario: LoadBalancing strategy (priority 10)

- **WHEN** strategy scores candidates
- **THEN** add random score (0-10) for load distribution

### Requirement: Group Management

The system SHALL support model grouping with selection strategies.

#### Scenario: Create group

- **WHEN** admin creates a group with models and strategy
- **THEN** store group configuration in database
- **AND** update group manager cache

#### Scenario: Group selection - priority

- **WHEN** group strategy is "priority"
- **THEN** select first available model in list order

#### Scenario: Group selection - round-robin

- **WHEN** group strategy is "round-robin"
- **THEN** cycle through models evenly

#### Scenario: Group selection - weighted

- **WHEN** group strategy is "weighted"
- **THEN** select models randomly based on configured weights

#### Scenario: Group selection - cost-optimal

- **WHEN** group strategy is "cost-optimal"
- **THEN** select model with lowest cost meeting requirements

#### Scenario: Group fallback

- **WHEN** primary model fails with configured trigger (rate_limit, timeout, server_error)
- **THEN** automatically retry with next model in group
- **AND** respect max_attempts limit

### Requirement: Routing Context

The system SHALL build routing context from request parameters.

#### Scenario: Build context from chat request

- **WHEN** LLMService builds routing context
- **THEN** extract task_type, estimated_tokens, stream requirement
- **AND** detect vision/tools requirements from messages

#### Scenario: Context with group override

- **WHEN** request includes routing.group parameter
- **THEN** use specified group instead of default

