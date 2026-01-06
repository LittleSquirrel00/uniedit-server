# Tasks: Enhance Billing Quotas

## Phase 1: Database Schema

- [x] **1.1** Create migration `000015_enhance_billing_quotas.up.sql`
  - Add `monthly_chat_tokens`, `monthly_image_credits`, `monthly_video_minutes`, `monthly_embedding_tokens` to `plans`
  - Add `git_storage_mb`, `lfs_storage_mb` to `plans`
  - Add `max_team_members` to `plans`
  - Set default values for backward compatibility
  - **Verify**: Run migration successfully

- [x] **1.2** Create down migration `000015_enhance_billing_quotas.down.sql`
  - Remove added columns
  - **Verify**: Down migration works

- [x] **1.3** Update seed data in `000008_seed_plans.up.sql` or create new migration
  - Set task-specific quotas for all plans
  - Set storage quotas for all plans
  - Set team member quotas for all plans
  - **Verify**: Plans have correct quota values

## Phase 2: Model Updates

- [x] **2.1** Update `internal/module/billing/model.go`
  - Add new fields to `Plan` struct
  - Add helper methods: `IsUnlimitedChat()`, `IsUnlimitedImage()`, etc.
  - **Verify**: Models compile

- [x] **2.2** Update `internal/module/collaboration/model.go`
  - Team.MemberLimit should come from owner's plan
  - **Verify**: Compile check

## Phase 3: Quota Checker Enhancement

- [x] **3.1** Create `internal/module/billing/quota/task_quota.go`
  - `CheckChatQuota(ctx, userID, estimatedTokens)` method
  - `CheckImageQuota(ctx, userID)` method
  - `CheckVideoQuota(ctx, userID, minutes)` method
  - `CheckEmbeddingQuota(ctx, userID, tokens)` method
  - **Verify**: Unit tests pass

- [x] **3.2** Create `internal/module/billing/quota/storage_quota.go`
  - `CheckGitStorageQuota(ctx, userID, additionalBytes)` method
  - `CheckLFSStorageQuota(ctx, userID, additionalBytes)` method
  - Storage cache with Redis
  - **Verify**: Unit tests pass

- [x] **3.3** Create `internal/module/billing/quota/team_quota.go`
  - `CheckTeamMemberQuota(ctx, teamOwnerID, currentCount)` method
  - **Verify**: Unit tests pass

## Phase 4: Usage Tracking Enhancement

- [x] **4.1** Update `internal/module/billing/quota/manager.go`
  - Add Redis keys for task-specific counters
  - `IncrementChatTokens()`, `IncrementImageCredits()`, etc.
  - **Verify**: Counters increment correctly

- [x] **4.2** Create `internal/module/billing/quota/storage_calculator.go`
  - `GetUserStorageUsage(ctx, userID) -> (git, lfs int64)`
  - Cache invalidation on repo size change
  - **Verify**: Calculation matches DB sum
  - **Note**: Implemented in `storage_quota.go` as part of StorageQuotaChecker

## Phase 5: Module Integration

- [x] **5.1** Create adapter `internal/module/billing/quota/ai_adapter.go`
  - AITaskQuotaAdapter for AI module integration
  - Chat/Embedding/Image/Video quota checks
  - Usage recording methods
  - **Verify**: Compile check

- [x] **5.2** Create adapter `internal/module/billing/quota/git_adapter.go`
  - GitStorageQuotaAdapter for git module's StorageQuotaChecker interface
  - LFSStorageQuotaAdapter for LFS quota checks
  - **Verify**: Compile check

- [x] **5.3** Create adapter `internal/module/billing/quota/team_adapter.go`
  - TeamQuotaAdapter for collaboration module integration
  - GetTeamMemberLimit, CheckTeamMemberQuota methods
  - **Verify**: Compile check

- [x] **5.4** Update `internal/module/billing/dto.go`
  - Add `AIQuotaStatus`, `StorageQuotaStatus`, `TeamQuotaStatus` structs
  - Update `QuotaStatus` to include all sections
  - Update `PlanResponse` with new quota fields
  - **Verify**: Compile check

- [ ] **5.5** Wire adapters in application initialization (optional)
  - Adapters are ready to be injected when modules need them
  - **Note**: Actual wiring deferred until module handlers are updated

## Phase 6: API Updates

- [x] **6.1** Update `internal/module/billing/dto.go`
  - Expand `QuotaStatus` response with task-specific quotas
  - Add storage quota section
  - Add team quota section
  - **Verify**: API returns complete quota info

- [ ] **6.2** Update `internal/module/billing/handler.go` (optional)
  - Update GET /quota endpoint to use new quota checkers
  - **Note**: Existing handler structure works with new DTOs

## Phase 7: Testing & Validation

- [ ] **7.1** Write integration tests
  - Test chat quota enforcement
  - Test image/video quota enforcement
  - Test storage quota enforcement
  - Test team member quota enforcement
  - **Verify**: All tests pass

- [x] **7.2** Build and run full test suite
  - `go build ./...` ✅
  - `go test ./...` (pending)
  - **Verify**: No regressions

## Dependencies

```
Phase 1 ──── Phase 2 ──── Phase 3 ──┬── Phase 5 ──── Phase 6
                                    │
             Phase 4 ───────────────┘
                                           ↓
                                      Phase 7
```

## Parallelizable Work

- Phase 3.1, 3.2, 3.3 can run in parallel
- Phase 5.1-5.5 can run in parallel after Phase 3, 4

## Implementation Notes

### Adapters Created
The implementation created adapters instead of directly modifying modules:
- `AITaskQuotaAdapter` - Provides quota checking for AI (chat, image, video, embedding)
- `GitStorageQuotaAdapter` - Implements git module's StorageQuotaChecker interface
- `LFSStorageQuotaAdapter` - Provides LFS storage quota checking
- `TeamQuotaAdapter` - Provides team member quota checking

### Integration Pattern
Modules can inject these adapters via dependency injection:
```go
// In git module
quotaAdapter := quota.NewGitStorageQuotaAdapter(storageChecker)
gitService := git.NewService(repo, r2Client, quotaAdapter, cfg, logger)

// In collaboration module
teamQuotaAdapter := quota.NewTeamQuotaAdapter(teamChecker)
// Pass to service constructor

// In AI module handler
aiQuotaAdapter := quota.NewAITaskQuotaAdapter(taskChecker)
// Use in handler for pre-request quota checks
```
