# Design: Provider Account Pool

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        AI Routing Layer                          │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    RoutingManager                         │   │
│  │  Route(ctx, req) → selects provider + model              │   │
│  └─────────────────────────┬────────────────────────────────┘   │
│                            │                                     │
│                            ▼                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │               AccountPoolManager (NEW)                    │   │
│  │  GetAccount(ctx, providerID) → returns available account │   │
│  │  MarkSuccess/Failure(ctx, accountID)                     │   │
│  └─────────────────────────┬────────────────────────────────┘   │
│                            │                                     │
│                            ▼                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    AccountScheduler                       │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │   │
│  │  │RoundRobin   │  │WeightedRandom│  │LeastConnections│   │   │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘   │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Data Layer                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                  provider_accounts                        │   │
│  │  id, provider_id, name, encrypted_key, weight, priority  │   │
│  │  health_status, last_check, failure_count, is_active     │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                  account_usage_stats                      │   │
│  │  account_id, date, requests, tokens, cost_usd            │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Component Design

### 1. ProviderAccount Model

```go
// ProviderAccount represents a single API key/account for a provider.
type ProviderAccount struct {
    ID                  uuid.UUID `gorm:"type:uuid;primaryKey"`
    ProviderID          uuid.UUID `gorm:"type:uuid;not null;index"`
    Name                string    `gorm:"not null"`
    EncryptedAPIKey     string    `gorm:"not null"`     // AES-256-GCM
    KeyPrefix           string    `gorm:"not null"`     // For identification (e.g., "sk-abc")

    // Scheduling
    Weight              int       `gorm:"default:1"`    // Higher = more traffic
    Priority            int       `gorm:"default:0"`    // Higher = preferred
    IsActive            bool      `gorm:"default:true"`

    // Health Monitoring
    HealthStatus        string    `gorm:"default:'healthy'"` // healthy, degraded, unhealthy
    LastHealthCheck     *time.Time
    ConsecutiveFailures int       `gorm:"default:0"`
    LastFailureAt       *time.Time

    // Rate Limits (per-account)
    RateLimitRPM        int       `gorm:"default:0"`    // 0 = use provider default
    RateLimitTPM        int       `gorm:"default:0"`
    DailyLimit          int       `gorm:"default:0"`

    // Usage Tracking (denormalized for fast access)
    TotalRequests       int64     `gorm:"default:0"`
    TotalTokens         int64     `gorm:"default:0"`
    TotalCostUSD        float64   `gorm:"default:0"`

    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

### 2. AccountPoolManager Interface

```go
// AccountPoolManager manages the pool of accounts for each provider.
type AccountPoolManager interface {
    // GetAccount returns an available account for the provider.
    // Returns ErrNoAvailableAccount if all accounts are unhealthy or rate-limited.
    GetAccount(ctx context.Context, providerID uuid.UUID) (*ProviderAccount, error)

    // MarkSuccess records a successful request for the account.
    MarkSuccess(ctx context.Context, accountID uuid.UUID, tokens int, costUSD float64) error

    // MarkFailure records a failed request for the account.
    MarkFailure(ctx context.Context, accountID uuid.UUID, err error) error

    // GetAccountStats returns usage statistics for an account.
    GetAccountStats(ctx context.Context, accountID uuid.UUID) (*AccountStats, error)

    // RefreshHealth triggers health checks for all accounts of a provider.
    RefreshHealth(ctx context.Context, providerID uuid.UUID) error
}
```

### 3. Scheduling Strategies

```go
// AccountScheduler defines the interface for account selection.
type AccountScheduler interface {
    // Select chooses an account from candidates based on strategy.
    Select(ctx context.Context, candidates []*ProviderAccount) (*ProviderAccount, error)
}

// Implementations:
// - RoundRobinScheduler: Cycles through accounts in order
// - WeightedRandomScheduler: Random selection weighted by account.Weight
// - LeastConnectionsScheduler: Selects account with fewest active requests
// - PriorityScheduler: Prefers higher priority, falls back on failure
```

### 4. Health Monitoring

```go
const (
    HealthStatusHealthy   = "healthy"
    HealthStatusDegraded  = "degraded"   // High latency or partial failures
    HealthStatusUnhealthy = "unhealthy"  // Circuit breaker open
)

// Health state transitions:
// healthy → degraded: 2+ consecutive failures OR latency > 3s
// degraded → unhealthy: 5+ consecutive failures
// unhealthy → degraded: After 30s cooldown, allow 1 test request
// degraded → healthy: 3 consecutive successes
```

### 5. API Key Management Enhancements

```go
// SystemAPIKey extensions
type SystemAPIKey struct {
    // ... existing fields ...

    // NEW: IP Whitelist
    AllowedIPs    pq.StringArray `gorm:"type:text[]"` // Empty = allow all

    // NEW: Auto-rotation
    RotateAfterDays   *int       // NULL = no auto-rotation
    LastRotatedAt     *time.Time

    // NEW: Audit tracking
    // (Separate table: api_key_audit_logs)
}

// APIKeyAuditLog tracks key usage and modifications.
type APIKeyAuditLog struct {
    ID         uuid.UUID
    APIKeyID   uuid.UUID
    Action     string    // "created", "used", "rotated", "disabled", "deleted"
    Details    map[string]any
    IPAddress  string
    UserAgent  string
    CreatedAt  time.Time
}
```

## Database Schema

### New Tables

```sql
-- Provider account pool
CREATE TABLE provider_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    encrypted_api_key TEXT NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,

    -- Scheduling
    weight INT DEFAULT 1,
    priority INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,

    -- Health
    health_status VARCHAR(20) DEFAULT 'healthy',
    last_health_check TIMESTAMP,
    consecutive_failures INT DEFAULT 0,
    last_failure_at TIMESTAMP,

    -- Rate limits
    rate_limit_rpm INT DEFAULT 0,
    rate_limit_tpm INT DEFAULT 0,
    daily_limit INT DEFAULT 0,

    -- Stats
    total_requests BIGINT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    total_cost_usd DECIMAL(12,6) DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_provider_account_name UNIQUE (provider_id, name)
);

CREATE INDEX idx_provider_accounts_provider ON provider_accounts(provider_id);
CREATE INDEX idx_provider_accounts_health ON provider_accounts(health_status, is_active);

-- Daily usage stats per account
CREATE TABLE account_usage_stats (
    id BIGSERIAL PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES provider_accounts(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    requests_count BIGINT DEFAULT 0,
    tokens_count BIGINT DEFAULT 0,
    cost_usd DECIMAL(12,6) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_account_date UNIQUE (account_id, date)
);

-- API key audit log
CREATE TABLE api_key_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_key_audit_key ON api_key_audit_logs(api_key_id);
CREATE INDEX idx_api_key_audit_time ON api_key_audit_logs(created_at);
```

### Modified Tables

```sql
-- Add IP whitelist and rotation to system_api_keys
ALTER TABLE system_api_keys
ADD COLUMN allowed_ips TEXT[] DEFAULT '{}',
ADD COLUMN rotate_after_days INT,
ADD COLUMN last_rotated_at TIMESTAMP;
```

## API Endpoints

### Provider Account Pool APIs (Admin)

```
POST   /admin/providers/:id/accounts      # Add account to pool
GET    /admin/providers/:id/accounts      # List accounts in pool
GET    /admin/providers/:id/accounts/:aid # Get account details
PATCH  /admin/providers/:id/accounts/:aid # Update account (weight, priority, active)
DELETE /admin/providers/:id/accounts/:aid # Remove account from pool

POST   /admin/providers/:id/accounts/:aid/check-health  # Trigger health check
GET    /admin/providers/:id/accounts/:aid/stats         # Get usage stats
```

### System API Key APIs (Enhanced)

```
GET    /api-keys/:id/audit-logs           # Get audit logs for key
PATCH  /api-keys/:id/ip-whitelist         # Update IP whitelist
POST   /api-keys/:id/schedule-rotation    # Schedule auto-rotation
```

## Integration Points

### 1. Routing Integration

```go
// In routing/manager.go
func (m *Manager) Route(ctx context.Context, req *Request) (*RouteResult, error) {
    // 1. Select provider + model (existing logic)
    provider, model := m.selectProviderModel(ctx, req)

    // 2. NEW: Get account from pool
    account, err := m.accountPool.GetAccount(ctx, provider.ID)
    if err != nil {
        return nil, fmt.Errorf("no available account: %w", err)
    }

    // 3. Return route result with account
    return &RouteResult{
        Provider:  provider,
        Model:     model,
        Account:   account,  // NEW field
        APIKey:    account.DecryptedKey,
    }, nil
}
```

### 2. Usage Recording Integration

```go
// After successful request
m.accountPool.MarkSuccess(ctx, account.ID, tokens, cost)

// After failed request
m.accountPool.MarkFailure(ctx, account.ID, err)
```

## Trade-offs

| Decision | Pros | Cons |
|----------|------|------|
| Store stats in main table (denormalized) | Fast reads, simple queries | Higher write load |
| Separate daily stats table | Historical analysis, lower main table writes | Extra join for current stats |
| Per-account rate limiting | Fine-grained control | Complexity in limit aggregation |
| Health check via real requests | Accurate status | Wastes tokens on failed accounts |

**Chosen approach**: Denormalized current stats + separate daily history table for balance of performance and analytics.
