# Proposal: Add Provider Account Pool

## Summary

Add a third-party AI provider account pool system with intelligent scheduling, health monitoring, and enhanced API key management.

## Motivation

Currently, each AI provider (OpenAI, Anthropic, etc.) can only have **one** API key configured. This creates several problems:

1. **Rate Limiting**: Single key hits rate limits easily under high load
2. **No Redundancy**: Key invalidation causes service outage
3. **No Load Balancing**: Cannot distribute requests across multiple accounts
4. **Cost Attribution**: Difficult to track costs across multiple billing accounts

## Scope

### In Scope

1. **Provider Account Pool** (NEW capability)
   - Multiple API keys per provider (account pooling)
   - Weighted round-robin scheduling with health awareness
   - Automatic failover on key failure
   - Per-account rate limiting and usage tracking
   - Health monitoring with circuit breaker pattern

2. **API Key Management** (MODIFY existing)
   - IP whitelist for system API keys
   - Automatic key rotation scheduling
   - API key audit logging

### Out of Scope

- Team-shared API keys (future feature)
- User-provided key pooling (use their own keys)
- Coupon/discount system for billing
- Multi-currency support

## Approach

Add a new `provider-account-pool` capability under the `ai` module that manages multiple accounts per provider. The routing module will be modified to request accounts from the pool instead of directly from the provider configuration.

## Dependencies

- **ai-provider**: Pool manages accounts per provider
- **ai-routing**: Route requests through pool scheduler
- **billing/usage**: Track usage per account for cost attribution

## Risks

| Risk | Mitigation |
|------|------------|
| Pool exhaustion under high load | Implement backpressure with 429 responses |
| Key leak if account is compromised | Quick disable mechanism and audit logging |
| Complexity in routing decisions | Clear scheduler interface abstraction |
