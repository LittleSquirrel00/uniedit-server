# Change: Refactor AI Module into Independent Modules

## Why

The current `internal/module/ai` module has grown to **10,697 lines of code** across 9 sub-packages, handling 7+ distinct responsibilities. This violates the Single Responsibility Principle and creates high coupling:

- **Provider management** (5,910 lines) is a central hub with 19 dependents
- **Media generation** (1,584 lines) is unrelated to LLM functionality
- **Task management** (2,918 lines) is generic infrastructure that could serve other modules
- The `adapter.Adapter` interface combines 4 unrelated concerns (Chat, Embed, HealthCheck, SupportsCapability)

## What Changes

### Module Restructuring

1. **Keep `ai` module** (LLM core, ~3,000 lines target)
   - Retain: adapter, routing, provider, group, llm, cache
   - Focus: Chat completion, embeddings, model routing

2. **Extract `media` module** (new, ~1,600 lines)
   - Move: media/ sub-package
   - Handler: /api/v1/media/*
   - Independent lifecycle and routing

3. **Extract `shared/task`** (new infrastructure, ~3,000 lines)
   - Move: task/ sub-package to shared infrastructure
   - Generic: Can be used by media, workflow, and future modules
   - Provides: TaskManager, Executor, Progress tracking

### Interface Refactoring

4. **Split `adapter.Adapter` interface**
   - `TextAdapter`: Chat, ChatStream
   - `EmbeddingAdapter`: Embed
   - `HealthChecker`: HealthCheck
   - `CapabilityProvider`: SupportsCapability

5. **Move interfaces to consumers**
   - Define `ProviderRegistry` interface in routing package
   - Define `BillingRecorder` interface in ai package

### Handler Decomposition

6. **Extract service layer from handlers**
   - Current: handler/admin.go (800 lines) mixes HTTP and business logic
   - Target: AdminService with separate HTTP handler

## Impact

### Affected Specs
- `ai-adapter` - Interface split
- `ai-llm-service` - Handler restructure
- `ai-media` - **MOVED** to new `media` capability
- `ai-task` - **MOVED** to new `shared-task` capability
- `ai-provider` - Minimal changes (keep in ai)
- `ai-routing` - Minimal changes (keep in ai)

### Affected Code
- `internal/module/ai/` - Major restructure
- `internal/module/media/` - **NEW**
- `internal/shared/task/` - **NEW**
- `internal/app/app.go` - Wire new modules
- `cmd/server/main.go` - Module registration

### Breaking Changes
- **BREAKING**: Handler routes for media move from `/api/v1/ai/media/*` to `/api/v1/media/*`
- **BREAKING**: Import paths change for media and task packages

### Migration Path
1. Phase 1: Extract shared/task (no API changes)
2. Phase 2: Extract media module (API route change)
3. Phase 3: Split adapter interfaces (internal refactor)
4. Phase 4: Extract service layer (internal refactor)
