# Tasks: Add Provider Account Pool

## Phase 1: Database & Models

- [x] **1.1** Create migration `000014_create_provider_account_pool.up.sql`
  - Create `provider_accounts` table
  - Create `account_usage_stats` table
  - Create `api_key_audit_logs` table
  - Add columns to `system_api_keys` (allowed_ips, rotate_after_days, last_rotated_at)
  - **Verify**: Run `go run ./cmd/migrate up` and check tables exist

- [x] **1.2** Create down migration `000014_create_provider_account_pool.down.sql`
  - Drop tables and columns in reverse order
  - **Verify**: Run `go run ./cmd/migrate down` and up again

- [x] **1.3** Define models in `internal/module/ai/provider/pool/model.go`
  - `ProviderAccount` struct with GORM tags
  - `AccountUsageStats` struct
  - `HealthStatus` constants
  - **Verify**: Models compile, GORM AutoMigrate works

## Phase 2: Account Pool Core

- [x] **2.1** Create `internal/module/ai/provider/pool/repository.go`
  - `Repository` interface: CRUD for accounts
  - `GetActiveAccountsByProvider(providerID)` query
  - `UpdateHealthStatus(accountID, status)` method
  - `RecordUsage(accountID, tokens, cost)` method
  - **Verify**: Unit tests for repository methods

- [x] **2.2** Create `internal/module/ai/provider/pool/scheduler.go`
  - `Scheduler` interface definition
  - `RoundRobinScheduler` implementation
  - `WeightedRandomScheduler` implementation
  - `PriorityScheduler` implementation
  - **Verify**: Unit tests for scheduler selection

- [x] **2.3** Create `internal/module/ai/provider/pool/health.go`
  - `HealthMonitor` struct
  - State transition logic (healthy → degraded → unhealthy)
  - Circuit breaker implementation
  - **Verify**: Unit tests for health state transitions

- [x] **2.4** Create `internal/module/ai/provider/pool/manager.go`
  - `Manager` struct implementing `AccountPoolManager` interface
  - `GetAccount()` with scheduler and health filtering
  - `MarkSuccess()` / `MarkFailure()` methods
  - Cache layer for active accounts
  - AES-256-GCM encryption for API keys
  - **Verify**: Unit tests for manager

- [x] **2.5** Create `internal/module/ai/provider/pool/dto.go`
  - Request/Response DTOs for API handlers
  - **Verify**: DTOs compile

## Phase 3: API Key Enhancements

- [x] **3.1** Update `internal/module/auth/model.go`
  - Add `AllowedIPs`, `RotateAfterDays`, `LastRotatedAt` to `SystemAPIKey`
  - Add `APIKeyAuditLog` model
  - Update `ToResponse()` method
  - **Verify**: Models compile

- [x] **3.2** Create `internal/module/auth/audit.go`
  - `AuditLogger` struct
  - `LogAction(keyID, action, metadata)` method
  - Repository for audit logs
  - **Verify**: Unit tests

- [x] **3.3** Create `internal/module/auth/ipvalidator.go`
  - `IPValidator` struct
  - `IsAllowed(clientIP, whitelist)` method
  - CIDR notation support
  - **Verify**: Unit tests for IP filtering

## Phase 4: Admin APIs

- [x] **4.1** Create `internal/module/ai/provider/pool/handler.go`
  - `POST /admin/providers/:provider_id/accounts` - Add account
  - `GET /admin/providers/:provider_id/accounts` - List accounts
  - `GET /admin/providers/:provider_id/accounts/:account_id` - Get account
  - `PATCH /admin/providers/:provider_id/accounts/:account_id` - Update account
  - `DELETE /admin/providers/:provider_id/accounts/:account_id` - Remove account
  - **Verify**: Manual API testing

- [x] **4.2** Add stats and health endpoints
  - `GET /admin/providers/:provider_id/accounts/:account_id/stats` - Usage stats
  - `POST /admin/providers/:provider_id/accounts/:account_id/check-health` - Trigger health check
  - **Verify**: Manual API testing

## Phase 5: Routing Integration

- [x] **5.1** Update `internal/module/ai/routing/context.go`
  - Add `AccountID` and `APIKey` fields to `Result` struct
  - **Verify**: Compiles

- [x] **5.2** Update `internal/module/ai/routing/manager.go`
  - Add `SetAccountPool()` method
  - Add `resolveAPIKey()` internal method
  - Add `MarkAccountSuccess()` / `MarkAccountFailure()` methods
  - **Verify**: Integration with pool manager

## Phase 6: App Integration

- [x] **6.1** Update `internal/shared/config/config.go`
  - Add `AccountPoolScheduler` config option
  - Add `AccountPoolCacheTTL` config option
  - Add `AccountPoolEncryptionKey` config option
  - Add defaults
  - **Verify**: Config loads correctly

- [x] **6.2** Update `internal/app/app.go`
  - Import pool package
  - Add `accountPoolHandler` and `accountPool` fields
  - Create `initAccountPoolModule()` function
  - Wire dependencies
  - Register admin routes
  - **Verify**: Server starts without errors

- [x] **6.3** Update `internal/module/ai/module.go`
  - Add `SetAccountPool()` method to wire pool to routing manager
  - **Verify**: Compiles

## Phase 7: Testing & Validation

- [x] **7.1** Build verification
  - Run `go build ./...`
  - **Verify**: No compilation errors

- [x] **7.2** Run existing tests
  - Run `go test ./internal/module/ai/routing/...`
  - Run `go test ./internal/module/auth/...`
  - **Verify**: All tests pass

- [ ] **7.3** Write integration tests (optional, future work)
  - Test account selection with multiple accounts
  - Test failover when account fails
  - Test circuit breaker behavior

## Dependencies

```
Phase 1 ──┬── Phase 2 ──┬── Phase 5 ──── Phase 6
          │             │
          └── Phase 3 ──┤
                        │
          Phase 4 ──────┘
                            ↓
                       Phase 7
```

## Parallelizable Work

- Phase 2 (Pool Core) and Phase 3 (API Key Enhancements) can run in parallel
- Phase 4 (Admin APIs) can start after Phase 2
- Phase 5 depends on Phase 2 completion

## Files Created/Modified

### New Files
- `migrations/000014_create_provider_account_pool.up.sql`
- `migrations/000014_create_provider_account_pool.down.sql`
- `internal/module/ai/provider/pool/model.go`
- `internal/module/ai/provider/pool/repository.go`
- `internal/module/ai/provider/pool/scheduler.go`
- `internal/module/ai/provider/pool/health.go`
- `internal/module/ai/provider/pool/manager.go`
- `internal/module/ai/provider/pool/dto.go`
- `internal/module/ai/provider/pool/handler.go`
- `internal/module/auth/audit.go`
- `internal/module/auth/ipvalidator.go`

### Modified Files
- `internal/module/auth/model.go`
- `internal/module/ai/routing/context.go`
- `internal/module/ai/routing/manager.go`
- `internal/module/ai/module.go`
- `internal/shared/config/config.go`
- `internal/app/app.go`
