## Context

The AI module has evolved into a monolithic package with 10,697 lines of code. Analysis reveals:

- **7+ distinct responsibilities** in one module
- **Provider package** is a central hub (19 dependents)
- **Adapter interface** combines 4 unrelated concerns
- **Media** and **Task** are orthogonal to LLM functionality

This refactoring addresses SOLID principle violations and prepares for future scalability.

## Goals / Non-Goals

### Goals
- Reduce AI module to ~3,000 lines (LLM core only)
- Enable independent development/deployment of media features
- Provide reusable task infrastructure for workflow and other modules
- Improve interface granularity following Interface Segregation Principle
- Reduce coupling between unrelated concerns

### Non-Goals
- Change external API behavior (beyond route migration for media)
- Add new features during refactoring
- Optimize performance (focus on architecture)
- Change database schema

## Decisions

### Decision 1: Module Boundaries

**What**: Split into 3 modules: `ai` (LLM), `media`, `shared/task`

**Why**: Each has distinct lifecycle, dependencies, and scaling requirements

**Alternatives considered**:
1. Keep as-is with internal boundaries → Doesn't solve coupling
2. Full microservice split → Over-engineering for current scale
3. Domain-driven packages within ai → Partial improvement, still coupled

### Decision 2: Shared Task Infrastructure

**What**: Place task system in `internal/shared/task/`

**Why**:
- Task management is generic infrastructure
- Workflow module will need similar capabilities
- Avoids duplication and ensures consistency

**Pattern**:
```go
// internal/shared/task/manager.go
type Manager interface {
    Submit(ctx context.Context, task *Task) (string, error)
    Get(ctx context.Context, id string) (*Task, error)
    Cancel(ctx context.Context, id string) error
    Subscribe(ctx context.Context, id string) (<-chan Progress, error)
}

// Executor registration
type Executor func(ctx context.Context, task *Task) error

func (m *manager) RegisterExecutor(taskType string, executor Executor)
```

### Decision 3: Interface Segregation for Adapter

**What**: Split `adapter.Adapter` into focused interfaces

**Current**:
```go
type Adapter interface {
    Type() provider.ProviderType
    Chat(ctx, req, model, provider) (*ChatResponse, error)
    ChatStream(ctx, req, model, provider) (<-chan ChatChunk, error)
    Embed(ctx, input, model, provider) (*EmbedResponse, error)
    HealthCheck(ctx, provider) error
    SupportsCapability(cap Capability) bool
}
```

**Proposed**:
```go
type TextAdapter interface {
    Chat(ctx, req, model, provider) (*ChatResponse, error)
    ChatStream(ctx, req, model, provider) (<-chan ChatChunk, error)
}

type EmbeddingAdapter interface {
    Embed(ctx, input, model, provider) (*EmbedResponse, error)
}

type HealthChecker interface {
    HealthCheck(ctx, provider) error
}

type CapabilityProvider interface {
    Type() provider.ProviderType
    SupportsCapability(cap Capability) bool
}

// Composite for backwards compatibility
type Adapter interface {
    TextAdapter
    EmbeddingAdapter
    HealthChecker
    CapabilityProvider
}
```

### Decision 4: Media Module Structure

**What**: Create `internal/module/media/` with standard module layout

**Structure**:
```
internal/module/media/
├── handler.go          # HTTP handlers (/api/v1/media/*)
├── service.go          # Business logic
├── adapter/            # Provider adapters (DALL-E, Runway)
│   ├── adapter.go      # Interface
│   ├── openai.go       # DALL-E
│   └── runway.go       # Video generation
├── model.go            # Domain models
├── dto.go              # Request/Response DTOs
├── repository.go       # Database access
└── module.go           # Wire module
```

**Routing**: All media endpoints under `/api/v1/media/*`

### Decision 5: Handler Service Extraction

**What**: Extract business logic from handlers into service layer

**Before**:
```go
// handler/admin.go (800 lines)
func (h *Handler) CreateProvider(c *gin.Context) {
    // Parse request
    // Validate
    // Encrypt API key
    // Save to database
    // Refresh cache
    // Return response
}
```

**After**:
```go
// service/admin_service.go
type AdminService struct {
    repo     ProviderRepository
    registry *ProviderRegistry
    crypto   CryptoService
}

func (s *AdminService) CreateProvider(ctx context.Context, req *CreateProviderRequest) (*Provider, error) {
    // Business logic here
}

// handler/admin.go (simplified)
func (h *Handler) CreateProvider(c *gin.Context) {
    var req CreateProviderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // Handle error
    }
    result, err := h.adminService.CreateProvider(c.Request.Context(), &req)
    // Handle response
}
```

## Target Architecture

```
internal/
├── module/
│   ├── ai/                      # LLM Core (~3,000 lines)
│   │   ├── handler/             # HTTP handlers
│   │   │   ├── chat.go          # /api/v1/ai/chat
│   │   │   ├── admin.go         # /api/v1/admin/ai/*
│   │   │   └── routes.go
│   │   ├── service/             # Business logic (NEW)
│   │   │   ├── llm_service.go
│   │   │   └── admin_service.go
│   │   ├── adapter/             # LLM adapters
│   │   │   ├── interface.go     # Split interfaces
│   │   │   ├── openai.go
│   │   │   ├── anthropic.go
│   │   │   └── generic.go
│   │   ├── routing/             # Routing strategies
│   │   ├── provider/            # Provider/model management
│   │   ├── group/               # Model groups
│   │   ├── cache/               # Embedding cache
│   │   └── module.go
│   │
│   └── media/                   # Media Generation (NEW ~1,600 lines)
│       ├── handler.go           # /api/v1/media/*
│       ├── service.go
│       ├── adapter/
│       │   ├── interface.go
│       │   ├── openai.go        # DALL-E
│       │   └── runway.go        # Video
│       ├── model.go
│       └── module.go
│
└── shared/
    └── task/                    # Task Infrastructure (NEW ~3,000 lines)
        ├── manager.go           # TaskManager interface
        ├── executor.go          # Executor registration
        ├── poller.go            # External task polling
        ├── progress.go          # Progress tracking
        ├── model.go             # Task, Progress models
        ├── repository.go        # Database access
        └── recovery.go          # Server restart recovery
```

## Dependency Graph (After Refactoring)

```
                    ┌─────────────────┐
                    │   app.go        │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
    ┌─────────┐        ┌───────────┐       ┌──────────┐
    │   ai    │        │   media   │       │ workflow │
    └────┬────┘        └─────┬─────┘       └────┬─────┘
         │                   │                   │
         │                   │                   │
         │                   └───────────────────┤
         │                           │           │
         ▼                           ▼           │
    ┌─────────┐              ┌──────────────┐    │
    │ billing │              │ shared/task  │◄───┘
    └─────────┘              └──────────────┘

Key:
- ai: No longer depends on media or task
- media: Uses shared/task for async operations
- workflow: Will use shared/task (future)
- Circular dependencies: NONE
```

## Risks / Trade-offs

### Risk 1: Breaking API Routes
- **Risk**: Media routes change from `/api/v1/ai/media/*` to `/api/v1/media/*`
- **Mitigation**:
  - Add redirect from old routes during transition
  - Document breaking change clearly
  - Version API if needed

### Risk 2: Migration Complexity
- **Risk**: Large refactoring may introduce bugs
- **Mitigation**:
  - Phase 1: Extract shared/task (no API changes)
  - Phase 2: Extract media module
  - Phase 3: Split interfaces (internal only)
  - Each phase has independent tests

### Risk 3: Import Path Changes
- **Risk**: Other modules importing ai sub-packages break
- **Mitigation**:
  - Update imports systematically
  - Use IDE refactoring tools
  - Clear compilation errors guide fixes

## Migration Plan

### Phase 1: Extract shared/task (Week 1)
1. Create `internal/shared/task/` package
2. Copy task logic, keeping original
3. Update media to use shared/task
4. Verify tests pass
5. Remove original task/ from ai

### Phase 2: Extract media module (Week 2)
1. Create `internal/module/media/`
2. Copy media logic with updated imports
3. Register new routes in app.go
4. Add redirect for old routes
5. Remove media/ from ai
6. Update client documentation

### Phase 3: Split adapter interfaces (Week 3)
1. Define new focused interfaces
2. Update concrete implementations
3. Create composite Adapter for compatibility
4. Update consumers to use specific interfaces

### Phase 4: Extract service layer (Week 4)
1. Create ai/service/ package
2. Extract business logic from handlers
3. Update handlers to call services
4. Add service-level tests

### Rollback Strategy
- Each phase is independently deployable
- Git branches per phase for easy rollback
- Feature flags if needed for gradual rollout

## Open Questions

1. **API Versioning**: Should we version media API as v2 or keep v1 with new path?
   - Recommendation: Keep v1, new path is sufficient

2. **Task Polling**: Should task polling be a separate service or goroutine pool?
   - Recommendation: Goroutine pool in shared/task, configurable concurrency

3. **Billing Integration**: Should media module directly call billing or go through shared interface?
   - Recommendation: Define BillingRecorder interface, inject into both ai and media
