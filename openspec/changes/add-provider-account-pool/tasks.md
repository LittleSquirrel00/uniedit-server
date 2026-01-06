# Tasks: Add Provider Account Pool

## Phase 1: Database & Models

- [ ] **1.1** Create migration `000014_create_provider_account_pool.up.sql`
  - Create `provider_accounts` table
  - Create `account_usage_stats` table
  - Create `api_key_audit_logs` table
  - Add columns to `system_api_keys` (allowed_ips, rotate_after_days, last_rotated_at)
  - **Verify**: Run `go run ./cmd/migrate up` and check tables exist

- [ ] **1.2** Create down migration `000014_create_provider_account_pool.down.sql`
  - Drop tables and columns in reverse order
  - **Verify**: Run `go run ./cmd/migrate down` and up again

- [ ] **1.3** Define models in `internal/module/ai/provider/pool/model.go`
  - `ProviderAccount` struct with GORM tags
  - `AccountUsageStats` struct
  - `HealthStatus` constants
  - **Verify**: Models compile, GORM AutoMigrate works

## Phase 2: Account Pool Core

- [ ] **2.1** Create `internal/module/ai/provider/pool/repository.go`
  - `Repository` interface: CRUD for accounts
  - `GetActiveAccountsByProvider(providerID)` query
  - `UpdateHealthStatus(accountID, status)` method
  - `RecordUsage(accountID, tokens, cost)` method
  - **Verify**: Unit tests for repository methods

- [ ] **2.2** Create `internal/module/ai/provider/pool/scheduler.go`
  - `Scheduler` interface definition
  - `RoundRobinScheduler` implementation
  - `WeightedRandomScheduler` implementation
  - **Verify**: Unit tests for scheduler selection

- [ ] **2.3** Create `internal/module/ai/provider/pool/health.go`
  - `HealthMonitor` struct
  - State transition logic (healthy → degraded → unhealthy)
  - Circuit breaker implementation
  - **Verify**: Unit tests for health state transitions

- [ ] **2.4** Create `internal/module/ai/provider/pool/manager.go`
  - `Manager` struct implementing `AccountPoolManager` interface
  - `GetAccount()` with scheduler and health filtering
  - `MarkSuccess()` / `MarkFailure()` methods
  - Cache layer for active accounts
  - **Verify**: Unit tests for manager

## Phase 3: API Key Enhancements

- [ ] **3.1** Update `internal/module/auth/model.go`
  - Add `AllowedIPs`, `RotateAfterDays`, `LastRotatedAt` to `SystemAPIKey`
  - Add `APIKeyAuditLog` model
  - **Verify**: Models compile

- [ ] **3.2** Create `internal/module/auth/audit.go`
  - `AuditLogger` interface
  - `LogAction(keyID, action, details, ip, userAgent)` method
  - **Verify**: Unit tests

- [ ] **3.3** Update `internal/module/auth/service.go`
  - Add IP whitelist validation in key verification
  - Add audit logging on key operations
  - Add auto-rotation scheduling logic
  - **Verify**: Unit tests for IP filtering

- [ ] **3.4** Update `internal/module/auth/handler.go`
  - Add `GET /api-keys/:id/audit-logs` endpoint
  - Add `PATCH /api-keys/:id/ip-whitelist` endpoint
  - Add `POST /api-keys/:id/schedule-rotation` endpoint
  - **Verify**: Manual API testing

## Phase 4: Admin APIs

- [ ] **4.1** Create `internal/module/ai/provider/pool/handler.go`
  - `POST /admin/providers/:id/accounts` - Add account
  - `GET /admin/providers/:id/accounts` - List accounts
  - `PATCH /admin/providers/:id/accounts/:aid` - Update account
  - `DELETE /admin/providers/:id/accounts/:aid` - Remove account
  - **Verify**: Manual API testing

- [ ] **4.2** Add stats and health endpoints
  - `GET /admin/providers/:id/accounts/:aid/stats` - Usage stats
  - `POST /admin/providers/:id/accounts/:aid/check-health` - Trigger health check
  - **Verify**: Manual API testing

## Phase 5: Routing Integration

- [ ] **5.1** Update `internal/module/ai/routing/manager.go`
  - Inject `AccountPoolManager` dependency
  - Call `GetAccount()` in routing flow
  - Return account info in `RouteResult`
  - **Verify**: Integration test with mock pool

- [ ] **5.2** Update routing to record account usage
  - Call `MarkSuccess/MarkFailure` after request completion
  - Pass account ID to usage recording
  - **Verify**: Usage stats update correctly

## Phase 6: App Integration

- [ ] **6.1** Update `internal/app/app.go`
  - Initialize account pool manager
  - Wire dependencies
  - Register admin routes
  - **Verify**: Server starts without errors

- [ ] **6.2** Add configuration options
  - Health check interval
  - Circuit breaker thresholds
  - Default scheduler strategy
  - **Verify**: Config loads correctly

## Phase 7: Testing & Validation

- [ ] **7.1** Write integration tests
  - Test account selection with multiple accounts
  - Test failover when account fails
  - Test circuit breaker behavior
  - **Verify**: All tests pass

- [ ] **7.2** Manual end-to-end testing
  - Add multiple accounts for a provider
  - Send requests and verify load balancing
  - Simulate failure and verify failover
  - **Verify**: System works as expected

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
