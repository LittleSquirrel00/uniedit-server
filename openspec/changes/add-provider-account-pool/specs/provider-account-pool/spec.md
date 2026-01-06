# provider-account-pool Specification

## Purpose

Manage multiple API key accounts per AI provider for load balancing, failover, and cost attribution.

## ADDED Requirements

### Requirement: Account Pool Management

The system SHALL manage a pool of API key accounts for each AI provider.

#### Scenario: Add account to pool

- **WHEN** admin adds account with provider_id, name, and api_key
- **THEN** the account is stored with encrypted api_key
- **AND** assigned default weight=1, priority=0, status=healthy

#### Scenario: List pool accounts

- **WHEN** admin requests accounts for a provider
- **THEN** return all accounts with health status and usage stats
- **AND** api_key is never exposed (only key_prefix)

#### Scenario: Update account

- **WHEN** admin updates account weight, priority, or active status
- **THEN** the account is updated
- **AND** routing decisions reflect changes immediately

#### Scenario: Remove account

- **WHEN** admin removes account from pool
- **THEN** the account is deleted
- **AND** pending requests using this account complete normally
- **AND** new requests use remaining accounts

### Requirement: Account Scheduling

The system SHALL select accounts from pool using configurable scheduling strategies.

#### Scenario: Round-robin selection

- **WHEN** scheduler strategy is "round_robin"
- **AND** pool has accounts A, B, C with same weight
- **THEN** requests cycle through A → B → C → A → ...

#### Scenario: Weighted selection

- **WHEN** scheduler strategy is "weighted"
- **AND** account A has weight=2, B has weight=1
- **THEN** A receives ~67% of requests, B receives ~33%

#### Scenario: Priority selection

- **WHEN** scheduler strategy is "priority"
- **AND** account A has priority=10, B has priority=5
- **THEN** A is always selected while healthy
- **AND** B is selected when A is unhealthy

#### Scenario: Skip unhealthy accounts

- **WHEN** selecting account from pool
- **AND** some accounts have health_status="unhealthy"
- **THEN** unhealthy accounts are excluded from selection

#### Scenario: No available accounts

- **WHEN** all accounts in pool are unhealthy or inactive
- **THEN** return error "no available accounts"
- **AND** respond with 503 Service Unavailable

### Requirement: Health Monitoring

The system SHALL monitor account health using circuit breaker pattern.

#### Scenario: Track request success

- **WHEN** request using account succeeds
- **THEN** reset consecutive_failures to 0
- **AND** if status was "degraded" and 3 consecutive successes, set status to "healthy"

#### Scenario: Track request failure

- **WHEN** request using account fails
- **THEN** increment consecutive_failures
- **AND** update last_failure_at

#### Scenario: Degrade account

- **WHEN** consecutive_failures >= 2 OR latency > 3 seconds
- **THEN** set health_status to "degraded"
- **AND** reduce selection weight temporarily

#### Scenario: Open circuit breaker

- **WHEN** consecutive_failures >= 5
- **THEN** set health_status to "unhealthy"
- **AND** exclude account from selection

#### Scenario: Circuit breaker recovery

- **WHEN** health_status is "unhealthy"
- **AND** 30 seconds have passed since last_failure_at
- **THEN** allow single test request (half-open state)
- **IF** test request succeeds
- **THEN** set health_status to "degraded"

### Requirement: Account Usage Tracking

The system SHALL track usage statistics per account.

#### Scenario: Record usage on success

- **WHEN** request completes successfully
- **THEN** increment account's total_requests, total_tokens
- **AND** add cost to total_cost_usd
- **AND** update daily stats in account_usage_stats

#### Scenario: Get account stats

- **WHEN** admin requests account statistics
- **THEN** return:
  - Total requests, tokens, cost (all time)
  - Daily breakdown for last 30 days
  - Current health status and consecutive failures

### Requirement: Per-Account Rate Limiting

The system SHALL support per-account rate limits.

#### Scenario: Account rate limit

- **WHEN** account has rate_limit_rpm > 0
- **AND** requests in last minute exceed rate_limit_rpm
- **THEN** exclude account from selection
- **AND** try other accounts in pool

#### Scenario: Daily limit

- **WHEN** account has daily_limit > 0
- **AND** requests today exceed daily_limit
- **THEN** exclude account from selection until next day

#### Scenario: Use provider defaults

- **WHEN** account has rate_limit_rpm = 0
- **THEN** use provider's default rate limits
