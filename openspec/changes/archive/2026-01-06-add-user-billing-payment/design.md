## Context

UniEdit Server currently only supports OAuth login (GitHub/Google). To enable monetization, we need:
1. Alternative registration method (email/password)
2. Subscription-based billing with usage quotas
3. Order and payment processing via Stripe

### Stakeholders
- End users: Register, subscribe, manage account
- Administrators: Manage users, view billing data
- System: Track usage, enforce quotas, process payments

### Constraints
- Must maintain backward compatibility with existing OAuth users
- Stripe is the only payment provider (for now)
- PostgreSQL + TimescaleDB for usage data storage

## Goals / Non-Goals

### Goals
- Enable email/password registration with email verification
- Implement subscription tiers (Free, Pro, Team, Enterprise)
- Quota enforcement on AI endpoints
- Stripe subscription and one-time payment support
- User self-service (account deletion, password reset)
- Admin user management (suspend/reactivate)

### Non-Goals
- Multi-tenancy / organization support (future)
- Multiple payment providers (Stripe only)
- Complex discount/coupon system
- Team member management

## Decisions

### D1: Module Structure
**Decision**: Create 4 separate modules (user, billing, order, payment) instead of a monolithic billing module.

**Rationale**:
- Clear separation of concerns (SOLID SRP)
- Independent scaling and testing
- Follows existing module patterns in codebase

**Alternatives considered**:
- Single "billing" module containing everything → Too large, violates SRP
- Merge user into auth → Auth is for authentication, user management is broader

### D2: User Model Extension vs. New Table
**Decision**: Extend existing `users` table with new fields rather than creating separate tables.

**Rationale**:
- Avoid JOIN overhead for common operations
- User identity is single entity
- Existing OAuth fields become nullable

**Migration**:
```sql
ALTER TABLE users
  ADD COLUMN status VARCHAR(50) DEFAULT 'active',
  ADD COLUMN password_hash TEXT,
  ADD COLUMN email_verified BOOLEAN DEFAULT false,
  ALTER COLUMN oauth_provider DROP NOT NULL,
  ALTER COLUMN oauth_id DROP NOT NULL;
```

### D3: Quota Storage
**Decision**: Use Redis for real-time quota counters, PostgreSQL for persistent limits.

**Rationale**:
- Redis provides atomic increment/decrement
- Survives pod restarts via persistence
- Daily reset via TTL

**Flow**:
```
Request → QuotaChecker middleware → Redis INCR → If OK, proceed
                                  → If exceeded, return 402
```

### D4: Usage Records in TimescaleDB
**Decision**: Store usage records in TimescaleDB hypertable for time-series analytics.

**Rationale**:
- Efficient time-based queries
- Automatic compression for older data
- Retention policies for data lifecycle

### D5: Stripe Subscription Model
**Decision**: Use Stripe Checkout for subscriptions, PaymentIntents for one-time charges.

**Rationale**:
- Stripe Checkout handles PCI compliance
- Built-in support for SCA (Strong Customer Authentication)
- Webhook-driven status updates

**Webhook Events**:
- `payment_intent.succeeded` → Mark order paid, add credits
- `customer.subscription.updated` → Sync subscription status
- `invoice.payment_failed` → Handle failed renewal

### D6: Order State Machine
**Decision**: Implement explicit state machine for order transitions.

**States**: `pending → paid → refunded` or `pending → canceled/failed`

**Rationale**:
- Prevents invalid transitions
- Clear audit trail
- Easier to reason about business logic

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Webhook delivery failure | Orders stuck in pending | Idempotent processing, manual reconciliation UI |
| Redis unavailable | Quota checks fail | Graceful degradation (allow request, log warning) |
| Email delivery issues | Users can't verify | Resend mechanism, admin verification override |
| Stripe API changes | Payment integration breaks | Pin SDK version, integration tests |

## Migration Plan

### Phase 1: Database Schema (P0)
1. Run migration 000004: Extend users table
2. Run migration 000005: Create billing tables (plans, subscriptions, usage_records)
3. Run migration 000006: Create order tables
4. Run migration 000007: Create payment tables
5. Run migration 000008: Seed initial plans

### Phase 2: Core Implementation (P0)
1. Implement user module (registration, verification)
2. Implement billing module (subscriptions, quotas)
3. Add quota middleware to AI endpoints

### Phase 3: Payment Integration (P1)
1. Implement Stripe provider
2. Implement order module
3. Implement payment module with webhooks

### Phase 4: Admin & Polish (P2)
1. Admin user management APIs
2. Invoice generation
3. Usage statistics dashboard APIs

### Rollback
- Database migrations have down scripts
- Feature flags for new registration flow
- Existing OAuth users unaffected

## Module Dependency Graph

```
                    ┌───────────┐
                    │   auth    │ (existing)
                    │  (JWT)    │
                    └─────┬─────┘
                          │ uses JWT/crypto
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
    ┌──────────┐    ┌──────────┐    ┌──────────┐
    │   user   │    │  billing │◄───│  order   │
    │          │    │          │    │          │
    └──────────┘    └────┬─────┘    └────┬─────┘
                         │               │
                         │    ┌──────────┘
                         ▼    ▼
                    ┌──────────┐
                    │ payment  │
                    │ (Stripe) │
                    └──────────┘
```

## Open Questions

1. **Email provider**: Which SMTP service to use? (Resend, SendGrid, SES)
   - *Suggestion*: Start with environment-configurable SMTP, abstract later

2. **Proration**: How to handle mid-cycle plan changes?
   - *Suggestion*: Let Stripe handle proration automatically

3. **Grace period**: How long after payment failure before suspension?
   - *Suggestion*: 7 days, configurable

4. **Usage aggregation frequency**: Real-time vs. batch?
   - *Suggestion*: Real-time write to TimescaleDB, batch aggregation for reports
