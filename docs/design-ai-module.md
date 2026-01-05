# AI 模块设计文档

> **版本**: v1.0 | **更新**: 2026-01-05
> **范围**: Provider、Adapter、Model、Routing、Grouping、Config、Task、Media
> **目标**: 完整的 AI 代理服务，支持多提供商、智能路由、媒体生成

---

## 一、架构概览

### 1.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        API Gateway                               │
│              (Gin Router + Middleware)                           │
└─────────────────────────────┬───────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│     LLM      │      │    Media     │      │    Task      │
│   Service    │      │   Service    │      │   Manager    │
│              │      │              │      │              │
│ • Chat       │      │ • Image Gen  │      │ • Submit     │
│ • Stream     │      │ • Video Gen  │      │ • Poll       │
│ • Embed      │      │ • Audio Gen  │      │ • Cancel     │
└──────┬───────┘      └──────┬───────┘      └──────┬───────┘
       │                     │                     │
       └─────────────────────┼─────────────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        ▼                    ▼                    ▼
┌──────────────┐      ┌──────────────┐     ┌──────────────┐
│   Routing    │      │   Provider   │     │    Config    │
│   Manager    │      │   Registry   │     │   Manager    │
│              │      │              │     │              │
│ • 6 策略链   │      │ • 适配器注册 │     │ • DB 配置    │
│ • 分组路由   │      │ • 健康监控   │     │ • 内存缓存   │
└──────────────┘      └──────────────┘     └──────────────┘
        │                    │                    │
        └────────────────────┼────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Adapter Layer                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │ OpenAI   │ │Anthropic │ │ Google   │ │ Generic  │ ...        │
│  │ Adapter  │ │ Adapter  │ │ Adapter  │ │ Adapter  │            │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 模块依赖关系

```
Database (底层)
     ↑
Provider/Model/Group Repository
     ↑
ProviderRegistry (内存缓存) ← AdapterRegistry
     ↑
RoutingManager (依赖 Registry)
     ↑
LLMService + MediaService + TaskManager
     ↑
HTTP Handlers
```

### 1.3 设计决策

| 问题 | 决策 |
|------|------|
| **配置存储** | 全部存数据库，启动时加载到内存 |
| **媒体生成** | 完整支持 LLM + 图片 + 视频 |
| **任务持久化** | 数据库持久化，支持历史查询和恢复 |
| **用户 Provider** | 仅平台配置，不支持用户自定义 |

---

## 二、目录结构

```
internal/module/ai/
├── adapter/                # 适配器层
│   ├── adapter.go          # Adapter 接口
│   ├── base.go             # BaseAdapter
│   ├── registry.go         # AdapterRegistry
│   ├── openai.go           # OpenAI 适配器
│   ├── anthropic.go        # Anthropic 适配器
│   ├── google.go           # Google 适配器
│   ├── generic.go          # 通用适配器
│   └── stream.go           # SSE 解析
│
├── provider/               # 提供商管理
│   ├── model.go            # Provider/Model 定义
│   ├── repository.go       # 数据库访问
│   ├── registry.go         # ProviderRegistry (内存缓存)
│   ├── health.go           # HealthMonitor
│   └── circuit_breaker.go  # CircuitBreaker
│
├── routing/                # 路由系统
│   ├── manager.go          # RoutingManager
│   ├── context.go          # RoutingContext
│   └── strategy/           # 6 种策略
│       ├── strategy.go     # Strategy 接口
│       ├── chain.go        # StrategyChain
│       ├── user_pref.go    # UserPreferenceStrategy
│       ├── health.go       # HealthFilterStrategy
│       ├── capability.go   # CapabilityFilterStrategy
│       ├── context_window.go # ContextWindowStrategy
│       ├── cost.go         # CostOptimizationStrategy
│       └── load_balance.go # LoadBalancingStrategy
│
├── group/                  # 分组管理
│   ├── model.go            # Group 定义
│   ├── repository.go       # 数据库访问
│   ├── manager.go          # GroupManager
│   └── selection.go        # 7 种选择策略
│
├── task/                   # 任务管理
│   ├── model.go            # Task 定义
│   ├── repository.go       # 数据库访问
│   ├── manager.go          # TaskManager
│   ├── executor.go         # TaskExecutor
│   └── pool.go             # ConcurrencyPool
│
├── media/                  # 媒体生成
│   ├── types.go            # 媒体类型定义
│   ├── service.go          # MediaService
│   ├── routing.go          # MediaRoutingManager
│   └── adapter/            # 媒体适配器
│       ├── adapter.go      # MediaAdapter 接口
│       ├── base.go         # BaseMediaAdapter
│       ├── openai.go       # DALL-E
│       ├── runway.go       # Runway 视频
│       └── registry.go     # MediaAdapterRegistry
│
├── llm_service.go          # LLMService (Chat/Stream/Embed)
├── handler.go              # HTTP Handler
└── dto.go                  # DTO 定义
```

---

## 三、数据库表设计

### 3.1 AI 提供商表

```sql
CREATE TABLE ai_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,       -- openai, anthropic, google, generic
    base_url TEXT NOT NULL,
    api_key TEXT NOT NULL,           -- AES-256 加密存储
    enabled BOOLEAN DEFAULT true,
    weight INT DEFAULT 1,
    priority INT DEFAULT 0,
    rate_limit JSONB,                -- {rpm, tpm, daily_limit}
    options JSONB,                   -- 额外配置
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_providers_type ON ai_providers(type);
CREATE INDEX idx_ai_providers_enabled ON ai_providers(enabled) WHERE enabled = true;
```

### 3.2 AI 模型表

```sql
CREATE TABLE ai_models (
    id VARCHAR(255) PRIMARY KEY,     -- 如 gpt-4o, claude-3-5-sonnet
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    capabilities TEXT[] NOT NULL,    -- {chat, stream, vision, tools}
    context_window INT NOT NULL,
    max_output_tokens INT NOT NULL,
    input_cost_per_1k DECIMAL(10, 6) NOT NULL,
    output_cost_per_1k DECIMAL(10, 6) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    options JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_models_provider ON ai_models(provider_id);
CREATE INDEX idx_ai_models_capabilities ON ai_models USING GIN(capabilities);
CREATE INDEX idx_ai_models_enabled ON ai_models(enabled) WHERE enabled = true;
```

### 3.3 AI 分组表

```sql
CREATE TABLE ai_groups (
    id VARCHAR(255) PRIMARY KEY,     -- 如 default, cost-optimized
    name VARCHAR(255) NOT NULL,
    task_type VARCHAR(50) NOT NULL,  -- chat, embedding, image, video
    models TEXT[] NOT NULL,          -- 模型 ID 列表
    strategy JSONB NOT NULL,         -- {type, weights?, max_cost_per_1k?}
    fallback JSONB,                  -- {enabled, max_attempts, trigger_on}
    required_capabilities TEXT[],
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_groups_task_type ON ai_groups(task_type);
```

### 3.4 AI 任务表

```sql
CREATE TABLE ai_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(50) NOT NULL,       -- chat, image_generation, video_generation
    status VARCHAR(50) NOT NULL,     -- pending, running, completed, failed, cancelled
    progress INT DEFAULT 0,
    input JSONB NOT NULL,            -- 任务输入
    output JSONB,                    -- 任务输出
    error JSONB,                     -- 错误信息
    external_task_id VARCHAR(255),   -- 外部平台任务 ID (如 Runway)
    provider_id UUID REFERENCES ai_providers(id),
    model_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_tasks_user ON ai_tasks(user_id, created_at DESC);
CREATE INDEX idx_ai_tasks_status ON ai_tasks(status) WHERE status IN ('pending', 'running');
CREATE INDEX idx_ai_tasks_external ON ai_tasks(external_task_id) WHERE external_task_id IS NOT NULL;
```

---

## 四、核心接口设计

### 4.1 Adapter 接口

```go
// internal/module/ai/adapter/adapter.go

// Capability 模型能力
type Capability string

const (
    CapabilityChat       Capability = "chat"
    CapabilityStream     Capability = "stream"
    CapabilityVision     Capability = "vision"
    CapabilityTools      Capability = "tools"
    CapabilityJSON       Capability = "json_mode"
    CapabilityEmbedding  Capability = "embedding"
    CapabilityImage      Capability = "image_generation"
    CapabilityVideo      Capability = "video_generation"
    CapabilityAudio      Capability = "audio_generation"
)

// Adapter LLM 适配器接口
type Adapter interface {
    // Type 返回适配器类型标识
    Type() string

    // Chat 非流式聊天
    Chat(ctx context.Context, req *ChatRequest, model *Model, provider *Provider) (*ChatResponse, error)

    // ChatStream 流式聊天
    ChatStream(ctx context.Context, req *ChatRequest, model *Model, provider *Provider) (<-chan *ChatChunk, error)

    // Embed 文本嵌入 (可选)
    Embed(ctx context.Context, input []string, model *Model, provider *Provider) ([][]float64, error)

    // HealthCheck 健康检查
    HealthCheck(ctx context.Context, provider *Provider) error

    // SupportsCapability 检查是否支持某能力
    SupportsCapability(cap Capability) bool
}

// ChatRequest 聊天请求
type ChatRequest struct {
    Model       string            `json:"model"`
    Messages    []*Message        `json:"messages"`
    MaxTokens   int               `json:"max_tokens,omitempty"`
    Temperature float64           `json:"temperature,omitempty"`
    TopP        float64           `json:"top_p,omitempty"`
    Stop        []string          `json:"stop,omitempty"`
    Tools       []*Tool           `json:"tools,omitempty"`
    ToolChoice  string            `json:"tool_choice,omitempty"`
    Stream      bool              `json:"stream,omitempty"`
    Metadata    map[string]any    `json:"metadata,omitempty"`
}

// Message 消息
type Message struct {
    Role       string         `json:"role"`
    Content    any            `json:"content"` // string 或 []ContentPart
    Name       string         `json:"name,omitempty"`
    ToolCallID string         `json:"tool_call_id,omitempty"`
    ToolCalls  []*ToolCall    `json:"tool_calls,omitempty"`
}

// ContentPart 多模态内容
type ContentPart struct {
    Type     string `json:"type"`               // text, image_url
    Text     string `json:"text,omitempty"`
    ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
    ID           string    `json:"id"`
    Model        string    `json:"model"`
    Message      *Message  `json:"message"`
    FinishReason string    `json:"finish_reason"`
    Usage        *Usage    `json:"usage"`
}

// ChatChunk 流式块
type ChatChunk struct {
    ID           string `json:"id"`
    Model        string `json:"model"`
    Delta        *Delta `json:"delta"`
    FinishReason string `json:"finish_reason,omitempty"`
}

// Delta 增量内容
type Delta struct {
    Role      string      `json:"role,omitempty"`
    Content   string      `json:"content,omitempty"`
    ToolCalls []*ToolCall `json:"tool_calls,omitempty"`
}

// Usage Token 用量
type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}
```

### 4.2 Provider/Model 模型

```go
// internal/module/ai/provider/model.go

// ProviderType 提供商类型
type ProviderType string

const (
    ProviderTypeOpenAI    ProviderType = "openai"
    ProviderTypeAnthropic ProviderType = "anthropic"
    ProviderTypeGoogle    ProviderType = "google"
    ProviderTypeAzure     ProviderType = "azure"
    ProviderTypeOllama    ProviderType = "ollama"
    ProviderTypeGeneric   ProviderType = "generic"
)

// Provider 提供商配置
type Provider struct {
    ID        uuid.UUID        `json:"id" gorm:"type:uuid;primary_key"`
    Name      string           `json:"name" gorm:"not null"`
    Type      ProviderType     `json:"type" gorm:"not null"`
    BaseURL   string           `json:"base_url" gorm:"not null"`
    APIKey    string           `json:"-" gorm:"not null"`  // 加密存储，不序列化
    Enabled   bool             `json:"enabled" gorm:"default:true"`
    Weight    int              `json:"weight" gorm:"default:1"`
    Priority  int              `json:"priority" gorm:"default:0"`
    RateLimit *RateLimitConfig `json:"rate_limit" gorm:"type:jsonb"`
    Options   map[string]any   `json:"options" gorm:"type:jsonb"`
    CreatedAt time.Time        `json:"created_at"`
    UpdatedAt time.Time        `json:"updated_at"`

    // Relations
    Models []*Model `json:"models,omitempty" gorm:"foreignKey:ProviderID"`
}

// Model 模型配置
type Model struct {
    ID              string         `json:"id" gorm:"primary_key"`
    ProviderID      uuid.UUID      `json:"provider_id" gorm:"type:uuid;not null"`
    Name            string         `json:"name" gorm:"not null"`
    Capabilities    pq.StringArray `json:"capabilities" gorm:"type:text[];not null"`
    ContextWindow   int            `json:"context_window" gorm:"not null"`
    MaxOutputTokens int            `json:"max_output_tokens" gorm:"not null"`
    InputCostPer1K  float64        `json:"input_cost_per_1k" gorm:"type:decimal(10,6)"`
    OutputCostPer1K float64        `json:"output_cost_per_1k" gorm:"type:decimal(10,6)"`
    Enabled         bool           `json:"enabled" gorm:"default:true"`
    Options         map[string]any `json:"options" gorm:"type:jsonb"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`

    // Relations
    Provider *Provider `json:"provider,omitempty" gorm:"foreignKey:ProviderID"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
    RPM        int `json:"rpm"`         // Requests per minute
    TPM        int `json:"tpm"`         // Tokens per minute
    DailyLimit int `json:"daily_limit"` // Daily request limit
}

// HasCapability 检查模型是否有某能力
func (m *Model) HasCapability(cap Capability) bool {
    for _, c := range m.Capabilities {
        if c == string(cap) {
            return true
        }
    }
    return false
}

// AverageCostPer1K 计算平均成本
func (m *Model) AverageCostPer1K() float64 {
    return (m.InputCostPer1K + m.OutputCostPer1K) / 2
}
```

### 4.3 Routing Strategy 接口

```go
// internal/module/ai/routing/strategy/strategy.go

// Strategy 路由策略接口
type Strategy interface {
    // Name 策略名称
    Name() string

    // Priority 优先级 (越高越先执行)
    Priority() int

    // Filter 过滤候选者
    Filter(ctx *RoutingContext, candidates []*ScoredCandidate) []*ScoredCandidate

    // Score 评分
    Score(ctx *RoutingContext, candidates []*ScoredCandidate) []*ScoredCandidate
}

// RoutingContext 路由上下文
type RoutingContext struct {
    TaskType           string            // chat, embedding, image, video
    EstimatedTokens    int               // 预估 token 数
    RequireStream      bool              // 是否需要流式
    RequireTools       bool              // 是否需要工具调用
    RequireVision      bool              // 是否需要视觉
    RequireJSON        bool              // 是否需要 JSON 模式
    MinContextWindow   int               // 最小上下文窗口
    MaxCostPer1K       float64           // 最大成本限制
    Optimize           string            // cost, quality, speed
    PreferredProviders []string          // 偏好的提供商
    ExcludedProviders  []string          // 排除的提供商
    PreferredModels    []string          // 偏好的模型
    ProviderHealth     map[string]bool   // 提供商健康状态
    Metadata           map[string]any    // 额外元数据
}

// ScoredCandidate 带分数的候选者
type ScoredCandidate struct {
    Provider       *Provider
    Model          *Model
    Score          float64
    ScoreBreakdown map[string]float64  // 各策略的分数
    Reasons        []string            // 选择原因
}

// RoutingResult 路由结果
type RoutingResult struct {
    Provider *Provider `json:"provider"`
    Model    *Model    `json:"model"`
    Score    float64   `json:"score"`
    Reason   string    `json:"reason"`
}
```

### 4.4 6 种路由策略

| 策略 | 优先级 | 作用 |
|------|--------|------|
| **UserPreferenceStrategy** | 100 | 应用用户偏好，排除指定提供商，首选 +20/+30 分 |
| **HealthFilterStrategy** | 90 | 过滤不健康的提供商（熔断器状态） |
| **CapabilityFilterStrategy** | 80 | 过滤不支持所需能力的模型 |
| **ContextWindowStrategy** | 70 | 过滤上下文窗口不足的模型，大窗口加分 |
| **CostOptimizationStrategy** | 50 | 过滤超预算模型，成本优化加分 |
| **LoadBalancingStrategy** | 10 | 随机加 0-10 分，实现负载均衡 |

```go
// 策略链执行流程
func (c *Chain) Route(ctx *RoutingContext, candidates []*ScoredCandidate) (*RoutingResult, error) {
    result := candidates

    // 按优先级执行各策略
    for _, strategy := range c.strategies {
        result = strategy.Filter(ctx, result)
        if len(result) == 0 {
            return nil, fmt.Errorf("no candidates after %s", strategy.Name())
        }
        result = strategy.Score(ctx, result)
    }

    // 选择最高分
    sort.Slice(result, func(i, j int) bool {
        return result[i].Score > result[j].Score
    })

    best := result[0]
    return &RoutingResult{
        Provider: best.Provider,
        Model:    best.Model,
        Score:    best.Score,
        Reason:   strings.Join(best.Reasons, "; "),
    }, nil
}
```

### 4.5 Task Manager 接口

```go
// internal/module/ai/task/manager.go

// TaskManager 任务管理器接口
type TaskManager interface {
    // Submit 提交任务
    Submit(ctx context.Context, userID uuid.UUID, input *TaskInput) (string, error)

    // Get 获取任务
    Get(ctx context.Context, id uuid.UUID) (*Task, error)

    // List 列出任务
    List(ctx context.Context, userID uuid.UUID, filter *TaskFilter) ([]*Task, error)

    // Cancel 取消任务
    Cancel(ctx context.Context, id uuid.UUID) error

    // WaitForCompletion 等待任务完成
    WaitForCompletion(ctx context.Context, id uuid.UUID, timeout time.Duration) (*Task, error)

    // OnProgress 订阅进度更新
    OnProgress(id uuid.UUID, callback func(*Task)) (unsubscribe func())

    // RegisterExecutor 注册执行器
    RegisterExecutor(taskType string, executor TaskExecutor)

    // RecoverPendingTasks 恢复未完成任务（启动时调用）
    RecoverPendingTasks(ctx context.Context) error
}

// TaskInput 任务输入
type TaskInput struct {
    Type     string         `json:"type"`
    Payload  map[string]any `json:"payload"`
    Priority int            `json:"priority,omitempty"`
    Timeout  time.Duration  `json:"timeout,omitempty"`
    Retry    *RetryConfig   `json:"retry,omitempty"`
}

// Task 任务
type Task struct {
    ID             uuid.UUID      `json:"id" gorm:"type:uuid;primary_key"`
    UserID         uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
    Type           string         `json:"type" gorm:"not null"`
    Status         TaskStatus     `json:"status" gorm:"not null"`
    Progress       int            `json:"progress" gorm:"default:0"`
    Input          map[string]any `json:"input" gorm:"type:jsonb;not null"`
    Output         map[string]any `json:"output,omitempty" gorm:"type:jsonb"`
    Error          *TaskError     `json:"error,omitempty" gorm:"type:jsonb"`
    ExternalTaskID string         `json:"external_task_id,omitempty"`
    ProviderID     *uuid.UUID     `json:"provider_id,omitempty" gorm:"type:uuid"`
    ModelID        string         `json:"model_id,omitempty"`
    CreatedAt      time.Time      `json:"created_at"`
    UpdatedAt      time.Time      `json:"updated_at"`
    CompletedAt    *time.Time     `json:"completed_at,omitempty"`
}

// TaskStatus 任务状态
type TaskStatus string

const (
    TaskStatusPending   TaskStatus = "pending"
    TaskStatusRunning   TaskStatus = "running"
    TaskStatusCompleted TaskStatus = "completed"
    TaskStatusFailed    TaskStatus = "failed"
    TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskExecutor 任务执行器
type TaskExecutor func(ctx context.Context, task *Task, onProgress func(int)) error
```

### 4.6 MediaAdapter 接口

```go
// internal/module/ai/media/adapter/adapter.go

// MediaGenerationType 媒体生成类型
type MediaGenerationType string

const (
    MediaTypeTextToImage  MediaGenerationType = "text-to-image"
    MediaTypeImageToImage MediaGenerationType = "image-to-image"
    MediaTypeTextToVideo  MediaGenerationType = "text-to-video"
    MediaTypeImageToVideo MediaGenerationType = "image-to-video"
    MediaTypeTextToAudio  MediaGenerationType = "text-to-audio"
    MediaTypeTextToMusic  MediaGenerationType = "text-to-music"
)

// MediaAdapter 媒体适配器接口
type MediaAdapter interface {
    // Type 返回适配器类型
    Type() string

    // SupportedTypes 返回支持的生成类型
    SupportedTypes() []MediaGenerationType

    // GenerateImage 生成图片
    GenerateImage(ctx context.Context, req *ImageRequest, model *Model, provider *Provider) (*MediaResult, error)

    // GenerateVideo 生成视频
    GenerateVideo(ctx context.Context, req *VideoRequest, model *Model, provider *Provider) (*MediaResult, error)

    // GetTaskStatus 获取异步任务状态
    GetTaskStatus(ctx context.Context, externalTaskID string, provider *Provider) (*MediaResult, error)

    // CancelTask 取消任务
    CancelTask(ctx context.Context, externalTaskID string, provider *Provider) error
}

// MediaResult 媒体生成结果
type MediaResult struct {
    ExternalTaskID string       `json:"external_task_id,omitempty"`
    Status         string       `json:"status"` // pending, processing, completed, failed
    Progress       int          `json:"progress,omitempty"`
    Outputs        []*MediaOutput `json:"outputs,omitempty"`
    Error          *MediaError  `json:"error,omitempty"`
}

// MediaOutput 媒体输出
type MediaOutput struct {
    URL       string `json:"url"`
    Type      string `json:"type"` // image, video, audio
    Format    string `json:"format,omitempty"`
    Width     int    `json:"width,omitempty"`
    Height    int    `json:"height,omitempty"`
    Duration  int    `json:"duration,omitempty"` // 秒
    FileSize  int64  `json:"file_size,omitempty"`
}
```

---

## 五、API 接口设计

### 5.1 Chat API

```yaml
POST /api/v1/ai/chat:
  description: 聊天补全
  headers:
    Authorization: Bearer {access_token}
  request:
    model: string              # 模型 ID 或 "auto"
    messages:
      - role: string           # system, user, assistant, tool
        content: string | array
    max_tokens: int            # 可选
    temperature: float         # 可选，0-2
    stream: bool               # 可选，默认 false
    tools: array               # 可选，工具定义
    routing:                   # 可选，路由配置
      group: string            # 分组 ID
      strategy: string         # priority, cost, quality
      fallback: bool
  response:
    id: string
    model: string
    choices:
      - index: int
        message:
          role: string
          content: string
          tool_calls: array
        finish_reason: string
    usage:
      prompt_tokens: int
      completion_tokens: int
      total_tokens: int
    _routing:
      provider_used: string
      model_used: string
      latency_ms: int
      cost_usd: float

POST /api/v1/ai/chat/stream:
  description: 流式聊天（SSE）
  headers:
    Authorization: Bearer {access_token}
    Accept: text/event-stream
  response:
    event: message
    data:
      id: string
      delta:
        role: string
        content: string
      finish_reason: string
```

### 5.2 Media API

```yaml
POST /api/v1/ai/images/generations:
  description: 图片生成
  request:
    prompt: string
    model: string              # 如 dall-e-3
    size: string               # 1024x1024, 1792x1024, 1024x1792
    quality: string            # standard, hd
    n: int                     # 生成数量
  response:
    task_id: string            # 任务 ID
    status: string
    outputs:
      - url: string
        type: image

POST /api/v1/ai/videos/generations:
  description: 视频生成
  request:
    prompt: string
    model: string              # 如 runway-gen3
    duration: int              # 秒
    aspect_ratio: string       # 16:9, 9:16, 1:1
    image_url: string          # 可选，图生视频
  response:
    task_id: string
    status: string             # pending

GET /api/v1/ai/tasks/{id}:
  description: 获取任务状态
  response:
    id: string
    type: string
    status: string
    progress: int
    output: object
    error: object
    created_at: timestamp
    completed_at: timestamp

DELETE /api/v1/ai/tasks/{id}:
  description: 取消任务
```

### 5.3 Admin API

```yaml
GET /api/v1/admin/ai/providers:
  description: 列出提供商
  response:
    items:
      - id: uuid
        name: string
        type: string
        enabled: bool
        models_count: int

POST /api/v1/admin/ai/providers:
  description: 创建提供商
  request:
    name: string
    type: string
    base_url: string
    api_key: string
    enabled: bool
    weight: int
    priority: int

GET /api/v1/admin/ai/models:
  description: 列出模型

POST /api/v1/admin/ai/models:
  description: 创建模型

GET /api/v1/admin/ai/groups:
  description: 列出分组

POST /api/v1/admin/ai/groups:
  description: 创建分组
```

---

## 六、Group 配置示例

### 6.1 默认聊天组

```json
{
  "id": "default",
  "name": "Default Chat",
  "task_type": "chat",
  "models": ["gpt-4o", "claude-3-5-sonnet", "deepseek-chat"],
  "strategy": {
    "type": "priority"
  },
  "fallback": {
    "enabled": true,
    "max_attempts": 3,
    "trigger_on": ["rate_limit", "timeout", "server_error"]
  }
}
```

### 6.2 成本优化组

```json
{
  "id": "cost-optimized",
  "name": "Cost Optimized",
  "task_type": "chat",
  "models": ["gpt-4o-mini", "deepseek-chat", "claude-3-haiku"],
  "strategy": {
    "type": "cost-optimal",
    "max_cost_per_1k": 0.001
  }
}
```

### 6.3 视觉任务组

```json
{
  "id": "vision",
  "name": "Vision Models",
  "task_type": "chat",
  "models": ["gpt-4o", "claude-3-5-sonnet"],
  "strategy": {
    "type": "round-robin"
  },
  "required_capabilities": ["vision"]
}
```

### 6.4 7 种选择策略

| 策略 | 说明 |
|------|------|
| **priority** | 按模型列表顺序，选择第一个可用 |
| **round-robin** | 轮询循环，均匀分配流量 |
| **weighted** | 按权重随机选择 |
| **cost-optimal** | 选择成本最低的模型 |
| **quality-optimal** | 选择质量评分最高的模型 |
| **latency-optimal** | 选择延迟最低的模型 |
| **capability-match** | 选择满足所有能力的第一个模型 |

---

## 七、实现步骤

### Phase 1: 数据模型和基础设施 (Day 1-2)

- [ ] 数据库迁移 (ai_providers, ai_models, ai_groups, ai_tasks)
- [ ] Repository 层实现
- [ ] 基础类型定义

### Phase 2: 适配器层 (Day 3-5)

- [ ] BaseAdapter 和 SSE 流式解析
- [ ] OpenAI Adapter (Chat/Stream/Vision/Tools)
- [ ] Anthropic Adapter
- [ ] Generic Adapter 和 AdapterRegistry

### Phase 3: Provider 管理 (Day 6-7)

- [ ] ProviderRegistry (数据库 + 内存缓存)
- [ ] HealthMonitor 和 CircuitBreaker

### Phase 4: 路由系统 (Day 8-10)

- [ ] RoutingManager 和策略链框架
- [ ] 6 种路由策略实现
- [ ] GroupManager 和 7 种选择策略

### Phase 5: 任务系统 (Day 11-13)

- [ ] TaskManager (数据库持久化)
- [ ] ConcurrencyPool 并发控制
- [ ] 任务恢复机制

### Phase 6: 媒体生成 (Day 14-16)

- [ ] MediaAdapter 接口和 BaseMediaAdapter
- [ ] OpenAI DALL-E 适配器
- [ ] Runway 视频适配器
- [ ] MediaService 和路由

### Phase 7: HTTP API (Day 17-18)

- [ ] Chat/Stream Handler
- [ ] Media Generation Handler
- [ ] Task Handler
- [ ] Admin Handler

### Phase 8: 测试 (Day 19-20)

- [ ] 单元测试
- [ ] 集成测试

---

## 八、设计原则

### SOLID 原则应用

| 原则 | 应用 |
|------|------|
| **S** - 单一职责 | 每个 Adapter 只处理一种协议 |
| **O** - 开闭原则 | 通过 Registry 注册新适配器 |
| **L** - 里氏替换 | 所有 Adapter 实现相同接口 |
| **I** - 接口隔离 | Adapter 接口最小化 |
| **D** - 依赖倒置 | Service 依赖接口不依赖实现 |

### Go 最佳实践

- 接口定义在使用方
- 使用 functional options 模式
- 错误包装 `fmt.Errorf("xxx: %w", err)`
- Context 贯穿调用链
- sync.RWMutex 保护共享状态
