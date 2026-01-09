# Phase 6-10 详细迁移方案

## 当前进度概览

```
Phase 0-5: 已完成 ✅
├── Phase 0: 准备工作 (目录结构、通用 Port/Model)
├── Phase 1: User 迁移 (覆盖率 80.4%)
├── Phase 2: Auth 迁移 (覆盖率 80.5%)
├── Phase 3: Billing 迁移 (覆盖率 81.0%)
├── Phase 4: Order 迁移 (覆盖率 88.5%)
└── Phase 5: Payment 迁移 (覆盖率 83.4%)

Phase 6-7: 已完成 ✅
├── Phase 6: AI 迁移 (复杂度: 高) ✅
└── Phase 7: Git 迁移 (复杂度: 高) ✅

Phase 8-10: 待完成 ⬜
├── Phase 8: Media 迁移 (复杂度: 中)
├── Phase 9: Collaboration 迁移 (复杂度: 中)
└── Phase 10: 清理优化
```

---

## Phase 6: AI 模块迁移

### 6.1 模块分析

**现有结构** (43 文件, 2000+ 行):
```
internal/module/ai/
├── adapter/           # 提供商适配器 (OpenAI, Anthropic, Generic)
├── cache/             # Embedding 缓存
├── group/             # 模型分组管理
├── handler/           # HTTP 处理器
├── llm/               # LLM 核心服务
├── provider/          # 提供商注册表
│   └── pool/          # 账户池管理
├── routing/           # 智能路由系统
├── mediaadapter.go    # 媒体适配器
├── module.go          # 模块入口
└── wire.go            # 依赖注入
```

**复杂度评估**:
- 策略模式: 6 个路由策略
- 注册表模式: Provider Registry, Adapter Registry
- 健康检查: 后台监控 + 断路器
- 账户池: 多账户调度

### 6.2 目标结构

```
internal/
├── domain/ai/
│   ├── domain.go           # AIDomain 接口 + 实现
│   ├── errors.go           # 领域错误
│   ├── routing.go          # 路由策略 (保留核心逻辑)
│   ├── strategy/           # 策略实现 (保留)
│   │   ├── capability.go
│   │   ├── cost.go
│   │   ├── health.go
│   │   ├── load_balance.go
│   │   └── user_pref.go
│   └── domain_test.go
│
├── model/ai.go             # AI 数据模型
│
├── port/
│   ├── inbound/ai.go       # HTTP 端口接口
│   └── outbound/ai.go      # 数据库、缓存、适配器端口
│
└── adapter/
    ├── inbound/gin/ai.go   # HTTP Handler
    └── outbound/
        ├── postgres/ai.go  # 数据库适配器
        ├── redis/ai.go     # 缓存适配器
        └── aivendor/       # AI 厂商适配器 (保留核心)
            ├── openai.go
            ├── anthropic.go
            ├── generic.go
            └── registry.go
```

### 6.3 迁移任务清单

#### P6.1 定义 AI Model

**产出文件**: `internal/model/ai.go`

```go
package model

// ProviderType represents AI provider type.
type ProviderType string

const (
    ProviderTypeOpenAI    ProviderType = "openai"
    ProviderTypeAnthropic ProviderType = "anthropic"
    ProviderTypeGoogle    ProviderType = "google"
    ProviderTypeAzure     ProviderType = "azure"
    ProviderTypeOllama    ProviderType = "ollama"
    ProviderTypeGeneric   ProviderType = "generic"
)

// AICapability represents model capability.
type AICapability string

const (
    CapabilityChat       AICapability = "chat"
    CapabilityStream     AICapability = "stream"
    CapabilityVision     AICapability = "vision"
    CapabilityTools      AICapability = "tools"
    CapabilityJSONMode   AICapability = "json_mode"
    CapabilityEmbedding  AICapability = "embedding"
    CapabilityImageGen   AICapability = "image_generation"
    CapabilityVideoGen   AICapability = "video_generation"
    CapabilityAudioGen   AICapability = "audio_generation"
)

// AIProvider represents an AI provider configuration.
type AIProvider struct {
    ID             uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
    Name           string          `json:"name"`
    Type           ProviderType    `json:"type"`
    BaseURL        string          `json:"base_url"`
    EncryptedKey   string          `json:"-"`  // 加密存储
    Enabled        bool            `json:"enabled"`
    Weight         int             `json:"weight"`
    Priority       int             `json:"priority"`
    RateLimitRPM   int             `json:"rate_limit_rpm"`
    RateLimitTPM   int             `json:"rate_limit_tpm"`
    CreatedAt      time.Time       `json:"created_at"`
    UpdatedAt      time.Time       `json:"updated_at"`
}

// AIModel represents an AI model.
type AIModel struct {
    ID              string         `json:"id" gorm:"primaryKey"`
    ProviderID      uuid.UUID      `json:"provider_id"`
    Name            string         `json:"name"`
    Capabilities    []string       `json:"capabilities" gorm:"type:jsonb"`
    ContextWindow   int            `json:"context_window"`
    MaxOutputTokens int            `json:"max_output_tokens"`
    InputCostPer1K  float64        `json:"input_cost_per_1k"`
    OutputCostPer1K float64        `json:"output_cost_per_1k"`
    Enabled         bool           `json:"enabled"`
}

// ProviderAccount represents a provider account in the pool.
type ProviderAccount struct {
    ID              uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey"`
    ProviderID      uuid.UUID    `json:"provider_id"`
    Name            string       `json:"name"`
    EncryptedAPIKey string       `json:"-"`
    KeyPrefix       string       `json:"key_prefix"`
    Weight          int          `json:"weight"`
    Priority        int          `json:"priority"`
    IsActive        bool         `json:"is_active"`
    HealthStatus    string       `json:"health_status"` // healthy, degraded, unhealthy
    RateLimitRPM    int          `json:"rate_limit_rpm"`
    RateLimitTPM    int          `json:"rate_limit_tpm"`
    TotalRequests   int64        `json:"total_requests"`
    TotalTokens     int64        `json:"total_tokens"`
    TotalCostUSD    float64      `json:"total_cost_usd"`
    CreatedAt       time.Time    `json:"created_at"`
    UpdatedAt       time.Time    `json:"updated_at"`
}

// ModelGroup represents a model group for routing.
type ModelGroup struct {
    ID                   string   `json:"id" gorm:"primaryKey"`
    Name                 string   `json:"name"`
    TaskType             string   `json:"task_type"` // chat, embedding, image, video, audio
    Models               []string `json:"models" gorm:"type:jsonb"`
    Strategy             string   `json:"strategy" gorm:"type:jsonb"`
    FallbackEnabled      bool     `json:"fallback_enabled"`
    FallbackMaxAttempts  int      `json:"fallback_max_attempts"`
    RequiredCapabilities []string `json:"required_capabilities" gorm:"type:jsonb"`
    Enabled              bool     `json:"enabled"`
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
    Model          string          `json:"model"`
    Messages       []ChatMessage   `json:"messages"`
    Temperature    *float64        `json:"temperature,omitempty"`
    MaxTokens      *int            `json:"max_tokens,omitempty"`
    TopP           *float64        `json:"top_p,omitempty"`
    Stream         bool            `json:"stream"`
    Tools          []Tool          `json:"tools,omitempty"`
    ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ChatMessage represents a chat message.
type ChatMessage struct {
    Role    string `json:"role"`
    Content any    `json:"content"` // string or []ContentPart
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
    ID        string       `json:"id"`
    Model     string       `json:"model"`
    Message   ChatMessage  `json:"message"`
    Usage     *Usage       `json:"usage,omitempty"`
    Routing   *RoutingInfo `json:"_routing,omitempty"`
}

// RoutingInfo contains routing metadata.
type RoutingInfo struct {
    ProviderUsed string  `json:"provider_used"`
    ModelUsed    string  `json:"model_used"`
    LatencyMs    int64   `json:"latency_ms"`
    CostUSD      float64 `json:"cost_usd"`
}
```

#### P6.2 定义 AI Outbound Port

**产出文件**: `internal/port/outbound/ai.go`

```go
package outbound

// AIProviderDatabasePort defines provider persistence.
type AIProviderDatabasePort interface {
    Create(ctx context.Context, provider *model.AIProvider) error
    FindByID(ctx context.Context, id uuid.UUID) (*model.AIProvider, error)
    FindAll(ctx context.Context) ([]*model.AIProvider, error)
    FindEnabled(ctx context.Context) ([]*model.AIProvider, error)
    Update(ctx context.Context, provider *model.AIProvider) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// AIModelDatabasePort defines model persistence.
type AIModelDatabasePort interface {
    Create(ctx context.Context, model *model.AIModel) error
    FindByID(ctx context.Context, id string) (*model.AIModel, error)
    FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIModel, error)
    FindByCapability(ctx context.Context, capability model.AICapability) ([]*model.AIModel, error)
    Update(ctx context.Context, model *model.AIModel) error
    Delete(ctx context.Context, id string) error
}

// ProviderAccountDatabasePort defines account pool persistence.
type ProviderAccountDatabasePort interface {
    Create(ctx context.Context, account *model.ProviderAccount) error
    FindByID(ctx context.Context, id uuid.UUID) (*model.ProviderAccount, error)
    FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.ProviderAccount, error)
    FindActiveByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.ProviderAccount, error)
    Update(ctx context.Context, account *model.ProviderAccount) error
    UpdateHealth(ctx context.Context, id uuid.UUID, status string) error
    IncrementUsage(ctx context.Context, id uuid.UUID, requests, tokens int64, cost float64) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// ModelGroupDatabasePort defines group persistence.
type ModelGroupDatabasePort interface {
    Create(ctx context.Context, group *model.ModelGroup) error
    FindByID(ctx context.Context, id string) (*model.ModelGroup, error)
    FindByTaskType(ctx context.Context, taskType string) ([]*model.ModelGroup, error)
    Update(ctx context.Context, group *model.ModelGroup) error
    Delete(ctx context.Context, id string) error
}

// AIProviderHealthCachePort defines health status caching.
type AIProviderHealthCachePort interface {
    GetHealth(ctx context.Context, providerID uuid.UUID) (bool, error)
    SetHealth(ctx context.Context, providerID uuid.UUID, healthy bool, ttl time.Duration) error
    GetAccountHealth(ctx context.Context, accountID uuid.UUID) (string, error)
    SetAccountHealth(ctx context.Context, accountID uuid.UUID, status string, ttl time.Duration) error
}

// EmbeddingCachePort defines embedding caching.
type EmbeddingCachePort interface {
    Get(ctx context.Context, key string) ([]float32, error)
    Set(ctx context.Context, key string, embedding []float32, ttl time.Duration) error
}

// AIVendorAdapterPort defines the interface for AI vendor adapters.
type AIVendorAdapterPort interface {
    Type() model.ProviderType
    SupportsCapability(cap model.AICapability) bool
    HealthCheck(ctx context.Context, provider *model.AIProvider) error
    Chat(ctx context.Context, req *model.ChatRequest, m *model.AIModel, p *model.AIProvider) (*model.ChatResponse, error)
    ChatStream(ctx context.Context, req *model.ChatRequest, m *model.AIModel, p *model.AIProvider) (<-chan *model.ChatChunk, error)
    Embed(ctx context.Context, input []string, m *model.AIModel, p *model.AIProvider) (*model.EmbedResponse, error)
}

// AIVendorRegistryPort defines vendor adapter registry.
type AIVendorRegistryPort interface {
    Register(adapter AIVendorAdapterPort)
    Get(providerType model.ProviderType) (AIVendorAdapterPort, error)
    GetForProvider(provider *model.AIProvider) (AIVendorAdapterPort, error)
}
```

#### P6.3 定义 AI Inbound Port

**产出文件**: `internal/port/inbound/ai.go`

```go
package inbound

// AIChatHttpPort defines chat HTTP handlers.
type AIChatHttpPort interface {
    Chat(c *gin.Context)
    ChatStream(c *gin.Context)
}

// AIEmbeddingHttpPort defines embedding HTTP handlers.
type AIEmbeddingHttpPort interface {
    Embed(c *gin.Context)
}

// AIProviderAdminHttpPort defines admin handlers for providers.
type AIProviderAdminHttpPort interface {
    ListProviders(c *gin.Context)
    CreateProvider(c *gin.Context)
    UpdateProvider(c *gin.Context)
    DeleteProvider(c *gin.Context)
    ListModels(c *gin.Context)
    SyncModels(c *gin.Context)
}

// AIAccountPoolHttpPort defines account pool handlers.
type AIAccountPoolHttpPort interface {
    ListAccounts(c *gin.Context)
    CreateAccount(c *gin.Context)
    UpdateAccount(c *gin.Context)
    DeleteAccount(c *gin.Context)
    GetAccountStats(c *gin.Context)
}
```

#### P6.4 创建 AI Domain

**产出文件**: `internal/domain/ai/domain.go`

**核心职责**:
1. Chat/ChatStream - 聊天请求处理
2. Embed - 文本嵌入
3. Route - 智能路由决策
4. HealthMonitor - 健康监控

**策略保留**:
- 从 `routing/strategy.go` 迁移 6 个策略到 `domain/ai/strategy/`
- 保持策略链模式

```go
package ai

// AIDomain defines AI service interface.
type AIDomain interface {
    // Chat sends a chat request and returns response.
    Chat(ctx context.Context, userID uuid.UUID, req *model.ChatRequest) (*model.ChatResponse, error)

    // ChatStream sends a streaming chat request.
    ChatStream(ctx context.Context, userID uuid.UUID, req *model.ChatRequest) (<-chan *model.ChatChunk, *model.RoutingInfo, error)

    // Embed generates embeddings for input texts.
    Embed(ctx context.Context, userID uuid.UUID, input []string, modelID string) (*model.EmbedResponse, error)

    // Route performs routing decision (for testing/debugging).
    Route(ctx context.Context, routingCtx *RoutingContext) (*RoutingResult, error)

    // StartHealthMonitor starts background health monitoring.
    StartHealthMonitor(ctx context.Context)

    // StopHealthMonitor stops health monitoring.
    StopHealthMonitor()
}
```

#### P6.5 创建 AI 适配器

**Postgres 适配器**:
- `internal/adapter/outbound/postgres/ai_provider.go`
- `internal/adapter/outbound/postgres/ai_model.go`
- `internal/adapter/outbound/postgres/ai_account.go`
- `internal/adapter/outbound/postgres/ai_group.go`

**Redis 适配器**:
- `internal/adapter/outbound/redis/ai_health.go`
- `internal/adapter/outbound/redis/ai_embedding.go`

**AI 厂商适配器** (重用现有代码):
- `internal/adapter/outbound/aivendor/openai.go`
- `internal/adapter/outbound/aivendor/anthropic.go`
- `internal/adapter/outbound/aivendor/generic.go`
- `internal/adapter/outbound/aivendor/registry.go`

**HTTP 适配器**:
- `internal/adapter/inbound/gin/ai_chat.go`
- `internal/adapter/inbound/gin/ai_admin.go`
- `internal/adapter/inbound/gin/ai_pool.go`

#### P6.6 迁移策略

**保留的核心逻辑**:
1. `routing/strategy.go` → `domain/ai/strategy/` (策略链实现)
2. `routing/manager.go` → `domain/ai/routing.go` (路由管理)
3. `provider/health.go` → `domain/ai/health.go` (健康监控)
4. `adapter/*.go` → `adapter/outbound/aivendor/` (厂商适配器)

**重构的逻辑**:
1. `provider/registry.go` → 拆分为 Port + Adapter
2. `llm/service.go` → `domain/ai/domain.go`
3. `handler/*.go` → `adapter/inbound/gin/ai_*.go`

#### P6.7 任务清单

| 序号 | 任务 | 产出文件 | 状态 |
|------|------|----------|------|
| P6.1 | 定义 AI Model | `internal/model/ai.go` | ⬜ |
| P6.2 | 定义 AI Outbound Port | `internal/port/outbound/ai.go` | ⬜ |
| P6.3 | 定义 AI Inbound Port | `internal/port/inbound/ai.go` | ⬜ |
| P6.4 | 创建 AI Domain | `internal/domain/ai/domain.go` | ⬜ |
| P6.5 | 迁移路由策略 | `internal/domain/ai/strategy/*.go` | ⬜ |
| P6.6 | 创建 Postgres 适配器 | `internal/adapter/outbound/postgres/ai_*.go` | ⬜ |
| P6.7 | 创建 Redis 适配器 | `internal/adapter/outbound/redis/ai_*.go` | ⬜ |
| P6.8 | 迁移厂商适配器 | `internal/adapter/outbound/aivendor/*.go` | ⬜ |
| P6.9 | 创建 HTTP 适配器 | `internal/adapter/inbound/gin/ai_*.go` | ⬜ |
| P6.10 | 更新 Domain Registry | `internal/domain/registry.go` | ⬜ |
| P6.11 | 编写 AI Domain 测试 | `internal/domain/ai/domain_test.go` | ⬜ |
| P6.12 | 集成验证 | - | ⬜ |

---

## Phase 7: Git 模块迁移

### 7.1 模块分析

**现有结构** (13 文件, 5079 行):
```
internal/module/git/
├── handler.go          # REST API 处理器 (921 行)
├── git_handler.go      # Git 协议处理器 (225 行)
├── service.go          # 业务逻辑 (809 行)
├── repository.go       # 数据访问 (465 行)
├── model.go            # 数据模型 (158 行)
├── dto.go              # DTO (206 行)
├── errors.go           # 错误定义 (50 行)
├── protocol/
│   └── smart_http.go   # Git Smart HTTP 协议 (474 行)
├── storage/
│   ├── r2.go           # R2 客户端 (317 行)
│   └── billy.go        # R2 文件系统 (575 行)
└── lfs/
    ├── batch.go        # LFS 批量 API (316 行)
    ├── lock.go         # LFS 锁定 API (376 行)
    └── storage.go      # LFS 存储 (187 行)
```

**复杂度评估**:
- Git Smart HTTP 协议实现
- LFS 批量操作和文件锁定
- R2/S3 文件系统 (billy 接口)
- 仓库权限控制

### 7.2 目标结构

```
internal/
├── domain/git/
│   ├── domain.go           # GitDomain 接口 + 实现
│   ├── errors.go           # 领域错误
│   ├── protocol.go         # Git 协议处理 (保留核心)
│   ├── lfs.go              # LFS 业务逻辑
│   └── domain_test.go
│
├── model/git.go            # Git 数据模型
│
├── port/
│   ├── inbound/git.go      # HTTP 端口接口
│   └── outbound/git.go     # 数据库、存储端口
│
└── adapter/
    ├── inbound/gin/
    │   ├── git_rest.go     # REST API Handler
    │   ├── git_protocol.go # Git 协议 Handler
    │   └── git_lfs.go      # LFS Handler
    └── outbound/
        ├── postgres/git.go # 数据库适配器
        └── r2/
            ├── client.go   # R2 客户端
            └── filesystem.go # Billy 文件系统
```

### 7.3 迁移任务清单

#### P7.1 定义 Git Model

**产出文件**: `internal/model/git.go`

```go
package model

// RepoType represents repository type.
type RepoType string

const (
    RepoTypeCode     RepoType = "code"
    RepoTypeWorkflow RepoType = "workflow"
    RepoTypeProject  RepoType = "project"
)

// RepoVisibility represents repository visibility.
type RepoVisibility string

const (
    VisibilityPublic  RepoVisibility = "public"
    VisibilityPrivate RepoVisibility = "private"
)

// RepoPermission represents repository permission level.
type RepoPermission string

const (
    PermissionRead  RepoPermission = "read"
    PermissionWrite RepoPermission = "write"
    PermissionAdmin RepoPermission = "admin"
)

// GitRepo represents a git repository.
type GitRepo struct {
    ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
    OwnerID       uuid.UUID      `json:"owner_id"`
    Name          string         `json:"name"`
    Slug          string         `json:"slug"`
    RepoType      RepoType       `json:"repo_type"`
    Visibility    RepoVisibility `json:"visibility"`
    Description   string         `json:"description"`
    DefaultBranch string         `json:"default_branch"`
    SizeBytes     int64          `json:"size_bytes"`
    LFSEnabled    bool           `json:"lfs_enabled"`
    LFSSizeBytes  int64          `json:"lfs_size_bytes"`
    StoragePath   string         `json:"storage_path"`
    StarsCount    int            `json:"stars_count"`
    ForksCount    int            `json:"forks_count"`
    ForkedFrom    *uuid.UUID     `json:"forked_from,omitempty"`
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
    PushedAt      *time.Time     `json:"pushed_at,omitempty"`
}

// RepoCollaborator represents a repository collaborator.
type RepoCollaborator struct {
    RepoID     uuid.UUID      `json:"repo_id" gorm:"primaryKey"`
    UserID     uuid.UUID      `json:"user_id" gorm:"primaryKey"`
    Permission RepoPermission `json:"permission"`
    CreatedAt  time.Time      `json:"created_at"`
}

// PullRequest represents a pull request.
type PullRequest struct {
    ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
    RepoID       uuid.UUID  `json:"repo_id"`
    Number       int        `json:"number"`
    Title        string     `json:"title"`
    Description  string     `json:"description"`
    SourceBranch string     `json:"source_branch"`
    TargetBranch string     `json:"target_branch"`
    Status       string     `json:"status"` // open, merged, closed
    AuthorID     uuid.UUID  `json:"author_id"`
    MergedBy     *uuid.UUID `json:"merged_by,omitempty"`
    MergedAt     *time.Time `json:"merged_at,omitempty"`
    ClosedAt     *time.Time `json:"closed_at,omitempty"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}

// LFSObject represents a Git LFS object.
type LFSObject struct {
    OID         string    `json:"oid" gorm:"primaryKey"` // SHA-256
    Size        int64     `json:"size"`
    StorageKey  string    `json:"storage_key"`
    ContentType string    `json:"content_type"`
    CreatedAt   time.Time `json:"created_at"`
}

// LFSLock represents a Git LFS file lock.
type LFSLock struct {
    ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
    RepoID   uuid.UUID `json:"repo_id"`
    Path     string    `json:"path"`
    OwnerID  uuid.UUID `json:"owner_id"`
    LockedAt time.Time `json:"locked_at"`
}
```

#### P7.2 定义 Git Outbound Port

**产出文件**: `internal/port/outbound/git.go`

```go
package outbound

// GitRepoDatabasePort defines repository persistence.
type GitRepoDatabasePort interface {
    Create(ctx context.Context, repo *model.GitRepo) error
    FindByID(ctx context.Context, id uuid.UUID) (*model.GitRepo, error)
    FindByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.GitRepo, error)
    FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*model.GitRepo, error)
    FindPublic(ctx context.Context, limit, offset int) ([]*model.GitRepo, error)
    Update(ctx context.Context, repo *model.GitRepo) error
    UpdateSize(ctx context.Context, id uuid.UUID, size int64) error
    UpdatePushedAt(ctx context.Context, id uuid.UUID) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// CollaboratorDatabasePort defines collaborator persistence.
type CollaboratorDatabasePort interface {
    Add(ctx context.Context, collab *model.RepoCollaborator) error
    Find(ctx context.Context, repoID, userID uuid.UUID) (*model.RepoCollaborator, error)
    FindByRepo(ctx context.Context, repoID uuid.UUID) ([]*model.RepoCollaborator, error)
    Update(ctx context.Context, collab *model.RepoCollaborator) error
    Remove(ctx context.Context, repoID, userID uuid.UUID) error
}

// PullRequestDatabasePort defines PR persistence.
type PullRequestDatabasePort interface {
    Create(ctx context.Context, pr *model.PullRequest) error
    FindByID(ctx context.Context, id uuid.UUID) (*model.PullRequest, error)
    FindByNumber(ctx context.Context, repoID uuid.UUID, number int) (*model.PullRequest, error)
    FindByRepo(ctx context.Context, repoID uuid.UUID, status *string, limit, offset int) ([]*model.PullRequest, error)
    Update(ctx context.Context, pr *model.PullRequest) error
    GetNextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}

// LFSObjectDatabasePort defines LFS object persistence.
type LFSObjectDatabasePort interface {
    Create(ctx context.Context, obj *model.LFSObject) error
    FindByOID(ctx context.Context, oid string) (*model.LFSObject, error)
    LinkToRepo(ctx context.Context, repoID uuid.UUID, oid string) error
    UnlinkFromRepo(ctx context.Context, repoID uuid.UUID, oid string) error
    GetRepoLFSSize(ctx context.Context, repoID uuid.UUID) (int64, error)
}

// LFSLockDatabasePort defines LFS lock persistence.
type LFSLockDatabasePort interface {
    Create(ctx context.Context, lock *model.LFSLock) error
    FindByID(ctx context.Context, id uuid.UUID) (*model.LFSLock, error)
    FindByPath(ctx context.Context, repoID uuid.UUID, path string) (*model.LFSLock, error)
    FindByRepo(ctx context.Context, repoID uuid.UUID, limit, offset int) ([]*model.LFSLock, error)
    Delete(ctx context.Context, id uuid.UUID) error
}

// GitStoragePort defines git repository storage.
type GitStoragePort interface {
    // InitBareRepo initializes a bare git repository.
    InitBareRepo(ctx context.Context, storagePath string) error

    // DeleteRepo deletes a repository from storage.
    DeleteRepo(ctx context.Context, storagePath string) error

    // GetFilesystem returns a billy.Filesystem for the repository.
    GetFilesystem(ctx context.Context, storagePath string) (billy.Filesystem, error)

    // PresignUpload generates a presigned upload URL for LFS.
    PresignUpload(ctx context.Context, key string, size int64, ttl time.Duration) (string, error)

    // PresignDownload generates a presigned download URL for LFS.
    PresignDownload(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// StorageQuotaCheckerPort defines storage quota checking.
type StorageQuotaCheckerPort interface {
    CheckQuota(ctx context.Context, userID uuid.UUID, requiredBytes int64) error
    GetUserTotalStorage(ctx context.Context, userID uuid.UUID) (int64, error)
}

// GitAuthenticatorPort defines git authentication.
type GitAuthenticatorPort interface {
    Authenticate(ctx context.Context, username, password string) (*model.User, error)
}
```

#### P7.3 定义 Git Inbound Port

**产出文件**: `internal/port/inbound/git.go`

```go
package inbound

// GitRepoHttpPort defines repository REST API handlers.
type GitRepoHttpPort interface {
    CreateRepo(c *gin.Context)
    GetRepo(c *gin.Context)
    ListRepos(c *gin.Context)
    ListPublicRepos(c *gin.Context)
    UpdateRepo(c *gin.Context)
    DeleteRepo(c *gin.Context)
    GetStorageStats(c *gin.Context)
}

// GitCollabHttpPort defines collaborator REST API handlers.
type GitCollabHttpPort interface {
    ListCollaborators(c *gin.Context)
    AddCollaborator(c *gin.Context)
    UpdateCollaborator(c *gin.Context)
    RemoveCollaborator(c *gin.Context)
}

// GitPRHttpPort defines pull request REST API handlers.
type GitPRHttpPort interface {
    CreatePR(c *gin.Context)
    GetPR(c *gin.Context)
    ListPRs(c *gin.Context)
    UpdatePR(c *gin.Context)
    MergePR(c *gin.Context)
}

// GitProtocolHttpPort defines Git protocol handlers.
type GitProtocolHttpPort interface {
    InfoRefs(c *gin.Context)
    UploadPack(c *gin.Context)
    ReceivePack(c *gin.Context)
}

// GitLFSHttpPort defines LFS API handlers.
type GitLFSHttpPort interface {
    BatchObjects(c *gin.Context)
    VerifyObject(c *gin.Context)
    CreateLock(c *gin.Context)
    ListLocks(c *gin.Context)
    VerifyLocks(c *gin.Context)
    DeleteLock(c *gin.Context)
}
```

#### P7.4 创建 Git Domain

**产出文件**: `internal/domain/git/domain.go`

```go
package git

// GitDomain defines Git service interface.
type GitDomain interface {
    // Repository operations
    CreateRepo(ctx context.Context, ownerID uuid.UUID, input *CreateRepoInput) (*model.GitRepo, error)
    GetRepo(ctx context.Context, ownerID uuid.UUID, slug string) (*model.GitRepo, error)
    GetRepoByID(ctx context.Context, id uuid.UUID) (*model.GitRepo, error)
    ListRepos(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*model.GitRepo, error)
    ListPublicRepos(ctx context.Context, limit, offset int) ([]*model.GitRepo, error)
    UpdateRepo(ctx context.Context, id uuid.UUID, input *UpdateRepoInput) (*model.GitRepo, error)
    DeleteRepo(ctx context.Context, id uuid.UUID) error

    // Access control
    CheckAccess(ctx context.Context, repoID, userID uuid.UUID, required model.RepoPermission) error
    CanAccess(ctx context.Context, repo *model.GitRepo, userID *uuid.UUID, required model.RepoPermission) bool

    // Collaborator operations
    AddCollaborator(ctx context.Context, repoID, userID uuid.UUID, perm model.RepoPermission) error
    ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*model.RepoCollaborator, error)
    UpdateCollaborator(ctx context.Context, repoID, userID uuid.UUID, perm model.RepoPermission) error
    RemoveCollaborator(ctx context.Context, repoID, userID uuid.UUID) error

    // Pull request operations
    CreatePR(ctx context.Context, repoID, authorID uuid.UUID, input *CreatePRInput) (*model.PullRequest, error)
    GetPR(ctx context.Context, repoID uuid.UUID, number int) (*model.PullRequest, error)
    ListPRs(ctx context.Context, repoID uuid.UUID, status *string, limit, offset int) ([]*model.PullRequest, error)
    UpdatePR(ctx context.Context, prID uuid.UUID, input *UpdatePRInput) (*model.PullRequest, error)
    MergePR(ctx context.Context, prID, mergerID uuid.UUID) error

    // Git protocol support
    GetFilesystem(ctx context.Context, repo *model.GitRepo) (billy.Filesystem, error)
    UpdatePushedAt(ctx context.Context, repoID uuid.UUID) error
    UpdateRepoSize(ctx context.Context, repoID uuid.UUID) error

    // Storage stats
    GetStorageStats(ctx context.Context, repoID uuid.UUID) (*StorageStats, error)
    GetUserStorageStats(ctx context.Context, userID uuid.UUID) (*StorageStats, error)
}

// LFSDomain defines LFS service interface.
type LFSDomain interface {
    // Batch operations
    ProcessBatch(ctx context.Context, repoID uuid.UUID, operation string, objects []*LFSBatchObject) ([]*LFSBatchResult, error)

    // Lock operations
    CreateLock(ctx context.Context, repoID, ownerID uuid.UUID, path string) (*model.LFSLock, error)
    ListLocks(ctx context.Context, repoID uuid.UUID, limit, offset int) ([]*model.LFSLock, error)
    VerifyLocks(ctx context.Context, repoID, ownerID uuid.UUID) (*LockVerifyResult, error)
    DeleteLock(ctx context.Context, lockID, ownerID uuid.UUID, force bool) error
}
```

#### P7.5 迁移策略

**保留的核心逻辑**:
1. `protocol/smart_http.go` → `domain/git/protocol.go` (Git 协议实现)
2. `storage/billy.go` → `adapter/outbound/r2/filesystem.go` (文件系统)
3. `lfs/*.go` → `domain/git/lfs.go` + `adapter/inbound/gin/git_lfs.go`

**重构的逻辑**:
1. `service.go` → `domain/git/domain.go`
2. `repository.go` → `adapter/outbound/postgres/git.go`
3. `handler.go` → `adapter/inbound/gin/git_rest.go`
4. `git_handler.go` → `adapter/inbound/gin/git_protocol.go`
5. `storage/r2.go` → `adapter/outbound/r2/client.go`

#### P7.6 任务清单

| 序号 | 任务 | 产出文件 | 状态 |
|------|------|----------|------|
| P7.1 | 定义 Git Model | `internal/model/git.go` | ⬜ |
| P7.2 | 定义 Git Outbound Port | `internal/port/outbound/git.go` | ⬜ |
| P7.3 | 定义 Git Inbound Port | `internal/port/inbound/git.go` | ⬜ |
| P7.4 | 创建 Git Domain | `internal/domain/git/domain.go` | ⬜ |
| P7.5 | 迁移 Git 协议 | `internal/domain/git/protocol.go` | ⬜ |
| P7.6 | 创建 LFS Domain | `internal/domain/git/lfs.go` | ⬜ |
| P7.7 | 创建 Postgres 适配器 | `internal/adapter/outbound/postgres/git.go` | ⬜ |
| P7.8 | 迁移 R2 存储适配器 | `internal/adapter/outbound/r2/*.go` | ⬜ |
| P7.9 | 创建 REST HTTP 适配器 | `internal/adapter/inbound/gin/git_rest.go` | ⬜ |
| P7.10 | 创建 Git 协议 HTTP 适配器 | `internal/adapter/inbound/gin/git_protocol.go` | ⬜ |
| P7.11 | 创建 LFS HTTP 适配器 | `internal/adapter/inbound/gin/git_lfs.go` | ⬜ |
| P7.12 | 更新 Domain Registry | `internal/domain/registry.go` | ⬜ |
| P7.13 | 编写 Git Domain 测试 | `internal/domain/git/domain_test.go` | ⬜ |
| P7.14 | 集成验证 | - | ⬜ |

---

## Phase 8: Media 模块迁移

### 8.1 模块分析

**现有结构** (7 文件, ~700 行):
```
internal/module/media/
├── model.go           # 数据模型 (132 行)
├── dto.go             # DTO
├── service.go         # 业务逻辑 (357 行)
├── registry.go        # 适配器注册表 (68 行)
├── registry_test.go
├── model_test.go
└── adapter/
    └── openai.go      # OpenAI 适配器
```

**复杂度评估**: 中等
- 图像生成 (同步)
- 视频生成 (异步任务)
- 适配器模式

### 8.2 目标结构

```
internal/
├── domain/media/
│   ├── domain.go           # MediaDomain 接口 + 实现
│   ├── errors.go           # 领域错误
│   └── domain_test.go
│
├── model/media.go          # Media 数据模型
│
├── port/
│   ├── inbound/media.go    # HTTP 端口接口
│   └── outbound/media.go   # 存储、任务、适配器端口
│
└── adapter/
    ├── inbound/gin/media.go
    └── outbound/
        └── mediavendor/
            ├── openai.go
            └── registry.go
```

### 8.3 任务清单

| 序号 | 任务 | 产出文件 | 状态 |
|------|------|----------|------|
| P8.1 | 定义 Media Model | `internal/model/media.go` | ⬜ |
| P8.2 | 定义 Media Outbound Port | `internal/port/outbound/media.go` | ⬜ |
| P8.3 | 定义 Media Inbound Port | `internal/port/inbound/media.go` | ⬜ |
| P8.4 | 创建 Media Domain | `internal/domain/media/domain.go` | ⬜ |
| P8.5 | 迁移 Media 适配器 | `internal/adapter/outbound/mediavendor/*.go` | ⬜ |
| P8.6 | 创建 HTTP 适配器 | `internal/adapter/inbound/gin/media.go` | ⬜ |
| P8.7 | 更新 Domain Registry | `internal/domain/registry.go` | ⬜ |
| P8.8 | 编写 Media Domain 测试 | `internal/domain/media/domain_test.go` | ⬜ |
| P8.9 | 集成验证 | - | ⬜ |

---

## Phase 9: Collaboration 模块迁移

### 9.1 模块分析

**现有结构** (7 文件, ~900 行):
```
internal/module/collaboration/
├── model.go           # Team, TeamMember, TeamInvitation
├── roles.go           # 权限定义
├── dto.go             # DTO
├── service.go         # 业务逻辑 (623 行)
├── repository.go      # 数据访问
├── errors.go
└── service_test.go
```

**复杂度评估**: 中等
- 团队管理
- 成员角色
- 邀请流程

### 9.2 目标结构

```
internal/
├── domain/collaboration/
│   ├── domain.go           # CollaborationDomain 接口 + 实现
│   ├── errors.go           # 领域错误
│   ├── roles.go            # 角色权限 (保留)
│   └── domain_test.go
│
├── model/collaboration.go  # Collaboration 数据模型
│
├── port/
│   ├── inbound/collaboration.go
│   └── outbound/collaboration.go
│
└── adapter/
    ├── inbound/gin/collaboration.go
    └── outbound/postgres/collaboration.go
```

### 9.3 任务清单

| 序号 | 任务 | 产出文件 | 状态 |
|------|------|----------|------|
| P9.1 | 定义 Collaboration Model | `internal/model/collaboration.go` | ⬜ |
| P9.2 | 定义 Collaboration Outbound Port | `internal/port/outbound/collaboration.go` | ⬜ |
| P9.3 | 定义 Collaboration Inbound Port | `internal/port/inbound/collaboration.go` | ⬜ |
| P9.4 | 创建 Collaboration Domain | `internal/domain/collaboration/domain.go` | ⬜ |
| P9.5 | 迁移角色权限 | `internal/domain/collaboration/roles.go` | ⬜ |
| P9.6 | 创建 Postgres 适配器 | `internal/adapter/outbound/postgres/collaboration.go` | ⬜ |
| P9.7 | 创建 HTTP 适配器 | `internal/adapter/inbound/gin/collaboration.go` | ⬜ |
| P9.8 | 更新 Domain Registry | `internal/domain/registry.go` | ⬜ |
| P9.9 | 编写 Collaboration Domain 测试 | `internal/domain/collaboration/domain_test.go` | ⬜ |
| P9.10 | 集成验证 | - | ⬜ |

---

## Phase 10: 清理与优化

### 10.1 删除旧模块代码

```bash
# 确保所有测试通过后
rm -rf internal/module/ai/
rm -rf internal/module/git/
rm -rf internal/module/media/
rm -rf internal/module/collaboration/

# 保留可能需要的共享代码
# rm -rf internal/module/auth/  # 如果已完全迁移
# rm -rf internal/module/user/  # 如果已完全迁移
```

### 10.2 更新 shared 目录

| 原位置 | 新位置 |
|--------|--------|
| `shared/config/` | `internal/config.go` |
| `shared/database/` | `internal/adapter/outbound/postgres/database.go` |
| `shared/cache/` | `internal/adapter/outbound/redis/cache.go` |
| `shared/storage/` | `internal/adapter/outbound/r2/client.go` |
| `shared/middleware/` | `internal/adapter/inbound/gin/middleware/` |
| `shared/crypto/` | `internal/adapter/outbound/crypto/` |
| `shared/errors/` | `internal/model/errors.go` |

### 10.3 更新 app.go

重写应用组装层，使用六边形架构模式。

### 10.4 更新文档

- [ ] 更新 `README.md`
- [ ] 更新 `CLAUDE.md`
- [ ] 生成 API 文档
- [ ] 更新架构图

---

## 依赖关系图

```
┌─────────────────────────────────────────────────────────────────┐
│                         Domain Layer                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────┐   ┌───────┐   ┌─────────┐   ┌───────┐   ┌─────────┐ │
│  │ User  │   │ Auth  │   │ Billing │   │ Order │   │ Payment │ │
│  └───┬───┘   └───┬───┘   └────┬────┘   └───┬───┘   └────┬────┘ │
│      │           │            │            │            │       │
│      └───────────┴────────────┴────────────┴────────────┘       │
│                              │                                   │
│  ┌───────┐   ┌───────┐   ┌───────┐   ┌───────────────┐         │
│  │  AI   │   │  Git  │   │ Media │   │ Collaboration │         │
│  └───┬───┘   └───┬───┘   └───┬───┘   └───────┬───────┘         │
│      │           │           │               │                   │
└──────┼───────────┼───────────┼───────────────┼───────────────────┘
       │           │           │               │
       ▼           ▼           ▼               ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Outbound Ports                             │
│  (Database, Cache, Storage, AI Vendors, Payment Providers...)  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| AI 模块复杂 | 迁移困难 | 保留策略链核心逻辑，仅重组结构 |
| Git 协议兼容性 | 功能回归 | 保留 smart_http.go 核心，增加集成测试 |
| 测试覆盖不足 | 迁移后 Bug | 先补充测试，再迁移 |
| 业务中断 | 用户影响 | 渐进式迁移，保持旧代码可用 |

---

## 时间线估算

| Phase | 复杂度 | 估算任务数 | 依赖 |
|-------|--------|-----------|------|
| Phase 6: AI | 高 | 12 | Phase 5 |
| Phase 7: Git | 高 | 14 | Phase 6 |
| Phase 8: Media | 中 | 9 | Phase 6 |
| Phase 9: Collaboration | 中 | 10 | Phase 5 |
| Phase 10: 清理 | 低 | 4 | Phase 6-9 |

**建议顺序**: P6 → P8 → P9 → P7 → P10

- P8/P9 相对独立，可与 P6 并行
- P7 最复杂，建议最后迁移
- P10 需等所有模块迁移完成
