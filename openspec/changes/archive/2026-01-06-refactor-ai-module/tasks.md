# Tasks: Refactor AI Module

## Phase 1: Extract shared/task Infrastructure ✅

### 1.1 Create shared/task Package
- [x] 1.1.1 Create `internal/shared/task/` directory structure
- [x] 1.1.2 Define `Manager` interface in `manager.go`
- [x] 1.1.3 Define `Executor` type and registration in `manager.go`
- [x] 1.1.4 Define `Task`, `Status`, `Error` models in `model.go`
- [x] 1.1.5 Implement `Repository` interface in `repository.go`
- [x] 1.1.6 Implement task polling via `ExternalTaskPoller`
- [x] 1.1.7 Implement progress subscription in `Manager`

### 1.2 Create Compatibility Layer
- [x] 1.2.1 Create `ai/task/compat.go` with type aliases
- [x] 1.2.2 Create `ai/task/manager_wrapper.go` wrapping shared manager
- [x] 1.2.3 Create `ai/task/repository_wrapper.go` wrapping shared repo
- [x] 1.2.4 Add conversion functions `ToSharedTask`/`FromSharedTask`

### 1.3 Database Migration
- [x] 1.3.1 Create migration `000016_refactor_tasks.up.sql`
- [x] 1.3.2 Rename table `ai_tasks` to `tasks`
- [x] 1.3.3 Rename column `user_id` to `owner_id`
- [x] 1.3.4 Add `metadata` column for generic task data

### 1.4 Verify and Test
- [x] 1.4.1 Run `go build ./...` - compilation successful
- [x] 1.4.2 Run `go test ./internal/shared/task/...`
- [x] 1.4.3 Run `go test ./internal/module/ai/task/...`
- [x] 1.4.4 Remove old `ai/task/model.go`, `repository.go`, `manager.go`

## Phase 2: Extract media Module ✅

### 2.1 Create media Module Structure
- [x] 2.1.1 Create `internal/module/media/` directory
- [x] 2.1.2 Create `model.go` with core types and Adapter interface
- [x] 2.1.3 Create `dto.go` with request/response types
- [x] 2.1.4 Create `registry.go` with AdapterRegistry
- [x] 2.1.5 Create `service.go` with media service
- [x] 2.1.6 Create `adapter/openai.go` with OpenAI DALL-E adapter

### 2.2 Add Tests
- [x] 2.2.1 Create `model_test.go` with model tests
- [x] 2.2.2 Create `registry_test.go` with registry tests

### 2.3 Verify
- [x] 2.3.1 Run `go build ./...` - compilation successful
- [x] 2.3.2 Run `go test ./internal/module/media/...`

**Note**: Kept existing `ai/media` for backward compatibility. New `media` module is independent and can be used for future migration.

## Phase 3: Split Adapter Interfaces ✅

### 3.1 Define New Interfaces
- [x] 3.1.1 Create `TypedAdapter` interface (Type)
- [x] 3.1.2 Create `CapabilityChecker` interface (SupportsCapability)
- [x] 3.1.3 Create `HealthChecker` interface (HealthCheck)
- [x] 3.1.4 Create `TextAdapter` interface (Chat, ChatStream)
- [x] 3.1.5 Create `EmbeddingAdapter` interface (Embed)
- [x] 3.1.6 Create composite `Adapter` embedding all interfaces

### 3.2 Update Implementations
- [x] 3.2.1 Add compile-time interface assertions to `openai.go`
- [x] 3.2.2 Add compile-time interface assertions to `anthropic.go`
- [x] 3.2.3 Add compile-time interface assertions to `generic.go`

### 3.3 Update Registry
- [x] 3.3.1 Add `GetTextAdapter` method
- [x] 3.3.2 Add `GetEmbeddingAdapter` method
- [x] 3.3.3 Add `GetHealthChecker` method

### 3.4 Add Tests
- [x] 3.4.1 Create `adapter_test.go` with interface tests
- [x] 3.4.2 Run `go test ./internal/module/ai/adapter/...`

## Phase 4: Extract Service Layer ✅

### 4.1 Define Service Interfaces
- [x] 4.1.1 Create `llm/interfaces.go` with service interfaces
- [x] 4.1.2 Define `ChatService` interface
- [x] 4.1.3 Define `EmbeddingService` interface
- [x] 4.1.4 Define composite `LLMService` interface

### 4.2 Update Handler
- [x] 4.2.1 Update `ChatHandler` to depend on interfaces
- [x] 4.2.2 Replace `llmService` field with `chatService` and `embeddingService`
- [x] 4.2.3 Update all method calls to use new fields

### 4.3 Add Tests
- [x] 4.3.1 Create `interfaces_test.go` with mock implementations
- [x] 4.3.2 Add interface compliance tests
- [x] 4.3.3 Run `go test ./internal/module/ai/llm/...`

### 4.4 Final Verification
- [x] 4.4.1 Run `go build ./...` - compilation successful
- [x] 4.4.2 Run `go test ./...` - all tests pass

## Phase 5: Documentation and Cleanup ✅

### 5.1 Update Task Documentation
- [x] 5.1.1 Update this tasks.md with completion status
- [x] 5.1.2 Archive this change with `openspec archive`

### 5.2 Summary of Changes
- Created `internal/shared/task/` - generic task infrastructure
- Created `internal/module/media/` - independent media module
- Split `Adapter` interface into 5 smaller interfaces (ISP)
- Defined `ChatService`, `EmbeddingService` interfaces (DIP)
- Handler now depends on interfaces, not implementations

---

## Future Work (Out of Scope)

The following tasks are deferred to a future change:
- Migrate existing `ai/media` usage to new `media` module
- Register new routes under `/api/v1/media/*`
- Add handler for new media module

These require API breaking changes and should be done in a separate change proposal.
