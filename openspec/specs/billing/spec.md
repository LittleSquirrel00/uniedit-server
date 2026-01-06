# billing Specification

## Purpose
TBD - created by archiving change add-user-billing-payment. Update Purpose after archive.
## Requirements
### Requirement: Plan Management
The system SHALL define subscription plans with quotas and pricing.

#### Scenario: List available plans
- **WHEN** user requests plan list
- **THEN** system returns all active plans with:
  - ID, name, description
  - Price (USD cents)
  - Billing cycle (monthly/yearly)
  - Monthly token limit
  - Daily request limit
  - Feature list

#### Scenario: Default plans
- **WHEN** system is initialized
- **THEN** the following plans exist:
  - free: 10K tokens/month, 100 requests/day, $0
  - pro_monthly: 500K tokens/month, 2000 requests/day, $20/month
  - pro_yearly: 500K tokens/month, 2000 requests/day, $200/year
  - team_monthly: 2M tokens/month, 10K requests/day, $50/month
  - team_yearly: 2M tokens/month, 10K requests/day, $500/year
  - enterprise: unlimited, custom pricing

### Requirement: Subscription Management
The system SHALL manage user subscriptions to plans.

#### Scenario: New user default subscription
- **WHEN** user completes registration
- **THEN** system creates subscription with free plan
- **AND** sets period_start to now, period_end to end of month

#### Scenario: Get subscription
- **WHEN** user requests their subscription
- **THEN** system returns subscription with plan details and quota status

#### Scenario: Upgrade subscription
- **WHEN** user upgrades to higher plan
- **THEN** system creates upgrade order
- **AND** upon payment, updates subscription immediately
- **AND** prorates charges via Stripe

#### Scenario: Downgrade subscription
- **WHEN** user downgrades to lower plan
- **THEN** system schedules change for next billing period
- **AND** sets cancel_at_period_end on current subscription

#### Scenario: Cancel subscription
- **WHEN** user cancels subscription
- **THEN** system sets cancel_at_period_end to true
- **AND** subscription remains active until period_end
- **AND** reverts to free plan after period_end

### Requirement: Quota Management
The system SHALL enforce usage quotas based on subscription plan.

#### Scenario: Check quota before request
- **WHEN** AI request is received
- **THEN** system checks:
  - Monthly token usage < plan limit
  - Daily request count < plan limit
- **IF** any limit exceeded
- **THEN** returns 402 Payment Required with quota_exceeded

#### Scenario: Unlimited quota
- **WHEN** plan has -1 for a limit (e.g., enterprise)
- **THEN** that limit is not enforced

#### Scenario: Get quota status
- **WHEN** user requests quota status
- **THEN** system returns:
  - tokens_used, tokens_limit, tokens_remaining
  - requests_today, requests_limit
  - reset_at (next period start)

#### Scenario: Quota reset
- **WHEN** billing period ends
- **THEN** monthly token counter resets to 0
- **WHEN** day changes (UTC)
- **THEN** daily request counter resets to 0

### Requirement: Usage Recording
The system SHALL record all AI usage for billing and analytics.

#### Scenario: Record usage
- **WHEN** AI request completes
- **THEN** system records:
  - user_id, timestamp
  - task_type (chat, image, video, embedding)
  - provider_id, model_id
  - input_tokens, output_tokens, total_tokens
  - cost_usd, latency_ms, success

#### Scenario: Failed request recording
- **WHEN** AI request fails
- **THEN** system still records usage with success=false
- **BUT** does not count against quota

#### Scenario: Usage aggregation
- **WHEN** user requests usage statistics
- **THEN** system aggregates by:
  - Time period (day, week, month)
  - Model
  - Task type
- **AND** returns totals for tokens, requests, cost

### Requirement: Credits Balance
The system SHALL maintain credits balance for pay-as-you-go usage.

#### Scenario: Add credits
- **WHEN** user completes top-up payment
- **THEN** system adds credit amount to subscription.credits_balance

#### Scenario: Deduct credits
- **WHEN** pay-per-use charge occurs (if applicable)
- **THEN** system deducts from credits_balance
- **AND** records transaction

#### Scenario: Get balance
- **WHEN** user requests balance
- **THEN** system returns credits_balance in USD cents

### Requirement: Subscription Status Sync
The system SHALL synchronize subscription status with Stripe.

#### Scenario: Status mapping
- **WHEN** Stripe subscription status changes
- **THEN** system maps to internal status:
  - trialing → trialing
  - active → active
  - past_due → past_due
  - canceled → canceled
  - incomplete → incomplete

#### Scenario: Past due handling
- **WHEN** subscription status is past_due
- **THEN** system continues to allow access for grace period (7 days)
- **AFTER** grace period, system downgrades to free plan

