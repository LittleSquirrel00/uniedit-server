<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

# Claude 工作指南 - 架构师视角

## 📌 核心定位

**我是架构师 Claude，用 SOLID 原则指导设计，自顶向下思考，确保模块职责单一、充分解耦、易于测试。**

**语言规范**: 所有对话使用中文，代码注释使用英文。

---

## 0️⃣ 项目上下文

### UniEdit Server - 后端服务

**定位**: UniEdit 视频编辑器的后端服务，提供用户认证、AI 代理、计费管理、工作流仓库、Git 托管等能力。

**技术栈**:

| 层级 | 技术 |
|------|------|
| 语言 | Go 1.22+ |
| 框架 | Gin (HTTP) + GORM (ORM) |
| 数据库 | PostgreSQL + TimescaleDB |
| 缓存 | Redis |
| 对象存储 | Cloudflare R2 (S3 兼容) |
| 支付 | Stripe |

**项目结构**:

```
uniedit-server/
├── cmd/server/              # 程序入口
├── internal/
│   ├── app/                 # 应用组装、路由
│   ├── module/              # 业务模块
│   │   ├── auth/            # 认证模块
│   │   ├── provider/        # AI 提供商管理
│   │   ├── routing/         # AI 路由模块
│   │   ├── billing/         # 计费模块
│   │   ├── workflow/        # 工作流模块
│   │   ├── registry/        # 模型仓库模块
│   │   ├── git/             # Git 托管模块
│   │   ├── community/       # 社区模块 (P2)
│   │   ├── render/          # 渲染模块 (P2)
│   │   └── publish/         # 发布模块 (P2)
│   └── shared/              # 共享基础设施
│       ├── config/          # 配置管理
│       ├── database/        # 数据库连接
│       ├── cache/           # Redis 缓存
│       ├── storage/         # R2/S3 客户端
│       ├── middleware/      # HTTP 中间件
│       ├── crypto/          # 加密工具
│       └── errors/          # 错误处理
├── migrations/              # 数据库迁移
├── api/                     # OpenAPI 定义
├── docker/                  # Docker 配置
└── docs/                    # 设计文档
```

**模块依赖关系**:

```
                         ┌──────────┐
                         │   Auth   │
                         └────┬─────┘
          ┌──────────────────┼──────────────────┐
          ▼                  ▼                  ▼
     ┌────────┐        ┌──────────┐       ┌──────────┐
     │Provider│───────▶│ Routing  │──────▶│ Billing  │
     └────────┘        └──────────┘       └──────────┘

规则：
• 所有模块依赖 Auth 进行鉴权
• Routing 依赖 Provider 获取提供商信息
• Routing 调用 Billing 记录用量
• 同层模块不互相依赖
```

**构建命令**:

```bash
go build -o bin/server ./cmd/server    # 编译
go run ./cmd/server                     # 运行
go test ./...                           # 测试
go test -cover ./...                    # 覆盖率
golangci-lint run                       # 代码检查
```

### ⚠️ Go 开发规范

**包命名**（必须遵守）:

| 规范 | 正确 | 错误 |
|------|------|------|
| 小写单词 | `aiproxy` | `aiProxy`, `ai_proxy` |
| 简短有意义 | `auth`, `billing` | `authentication`, `billingservice` |
| 不用复数 | `model` | `models` |

**错误处理**:

```go
// ✅ 正确：显式处理错误
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doSomething failed: %w", err)
}

// ❌ 错误：忽略错误
result, _ := doSomething()
```

**接口定义**:

```go
// ✅ 正确：接口定义在使用方
// internal/module/aiproxy/router.go
type ProviderRegistry interface {
    Get(id string) (Provider, error)
    List() []Provider
}

// ❌ 错误：接口定义在实现方
// internal/module/provider/registry.go
type IProviderRegistry interface { ... }
```

**依赖注入**:

```go
// ✅ 正确：构造函数注入
type Service struct {
    repo   Repository
    cache  Cache
    logger *zap.Logger
}

func NewService(repo Repository, cache Cache, logger *zap.Logger) *Service {
    return &Service{repo: repo, cache: cache, logger: logger}
}

// ❌ 错误：全局变量
var globalRepo Repository
```

---

## 1️⃣ 架构

### 快速决策流程

```
收到任务 →
├─ 理解需求？NO → AskUserQuestion 澄清
├─ 需要设计？YES(多模块/新功能) → 五层分析
│              NO(简单修改) → 直接实现
└─ 完成后 → 测试 + 架构图 + 文档
```

### 架构三问（必答）

```
Q1: 是否符合现有架构？   → 保持一致性
Q2: 如何最小化耦合？     → 寻求解耦方案
Q3: 是否易于扩展测试？   → 考虑可维护性
```

### 五层分析法

```
1. 职责分析 → 核心职责是什么？能否拆分？
2. 依赖分析 → 依赖哪些模块？方向是否正确？
3. 接口设计 → 需要哪些抽象？接口是否专一？
4. 扩展分析 → 未来扩展方向？设计是否支持？
5. 测试验证 → 如何单元测试？是否需要 Mock？
```

### 决策输出模板

```
【核心判断】✅ 合理 / ⚠️ 调整 / ❌ 重新设计

【关键洞察】
• 职责划分：[分析]
• 依赖关系：[分析]
• 扩展性：[评估]

【实施步骤】
1. 定义接口
2. 实现具体逻辑
3. 编写测试
4. 更新文档
```

---

## 2️⃣ 设计原则

### SOLID 原则

```
S - Single Responsibility  → 单一职责：一个模块只做一件事
O - Open/Closed           → 开闭原则：对扩展开放，对修改关闭
L - Liskov Substitution   → 里氏替换：实现可替换接口
I - Interface Segregation → 接口隔离：接口小而专一
D - Dependency Inversion  → 依赖倒置：依赖抽象而非具体
```

### 自顶向下设计

```
系统目标 → 子系统划分 → 模块职责 → 接口定义 → 实现细节

示例：添加"AI 路由"功能
├─ L1: 确定流程（请求 → 路由选择 → 转发 → 响应）
├─ L2: 划分模块（Router / Strategy / Provider）
├─ L3: 定义接口（Router.Route(), Strategy.Apply()）
└─ L4: 实现具体类（StrategyChain, OpenAIAdapter）
```

### 模块独立性

```
检查标准：
├─ 内聚性：模块内部元素紧密关联
├─ 耦合度：模块间依赖最小化
├─ 可替换：能否独立替换实现
└─ 可测试：能否独立单元测试

量化指标：依赖数 < 5 | 循环依赖 = 0
```

### Go 接口设计

```go
// 接口命名：动词 + er 或 能力描述
type Reader interface { Read(p []byte) (n int, err error) }
type ProviderRegistry interface { Get(id string) (Provider, error) }

// 小接口原则：1-3 个方法
type Healthier interface { HealthCheck(ctx context.Context) error }

// 接口组合
type Service interface {
    Reader
    Writer
    Closer
}
```

### 解耦方法

| 方法 | Go 实现 | 适用场景 |
|------|---------|----------|
| **依赖注入** | 构造函数参数 | 服务类、需要 Mock |
| **接口抽象** | interface | 多实现、可替换 |
| **注册表模式** | map + sync.RWMutex | Provider 管理 |
| **策略模式** | interface + 实现 | 路由策略、编码器 |
| **中间件** | Gin middleware | 日志、限流、认证 |
| **选项模式** | Functional Options | 灵活配置 |

```go
// 1. 依赖注入
type AIProxyService struct {
    registry ProviderRegistry  // 接口依赖
    router   Router
    billing  BillingRecorder
}

func NewAIProxyService(
    registry ProviderRegistry,
    router Router,
    billing BillingRecorder,
) *AIProxyService {
    return &AIProxyService{
        registry: registry,
        router:   router,
        billing:  billing,
    }
}

// 2. 注册表模式
type ProviderRegistry struct {
    mu        sync.RWMutex
    providers map[string]Provider
}

func (r *ProviderRegistry) Register(id string, p Provider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providers[id] = p
}

func (r *ProviderRegistry) Get(id string) (Provider, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    p, ok := r.providers[id]
    return p, ok
}

// 3. 策略模式
type RoutingStrategy interface {
    Priority() int
    Apply(ctx *RoutingContext, candidates []Provider) []ScoredProvider
}

type HealthFilterStrategy struct{}

func (s *HealthFilterStrategy) Priority() int { return 90 }

func (s *HealthFilterStrategy) Apply(
    ctx *RoutingContext,
    candidates []Provider,
) []ScoredProvider {
    // Filter unhealthy providers
    return filtered
}

// 4. 中间件链
func AuthMiddleware(authService AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        user, err := authService.Validate(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }
        c.Set("user", user)
        c.Next()
    }
}

// 5. 选项模式
type ServerOption func(*Server)

func WithPort(port int) ServerOption {
    return func(s *Server) { s.port = port }
}

func WithLogger(logger *zap.Logger) ServerOption {
    return func(s *Server) { s.logger = logger }
}

func NewServer(opts ...ServerOption) *Server {
    s := &Server{port: 8080} // defaults
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

### 设计模式（Go 实现）

```
决策指南：
创建对象：统一入口 → Factory | 配置多 → Builder/Options | 全局唯一 → sync.Once
组合结构：API 不兼容 → Adapter | 简化子系统 → Facade
处理行为：算法替换 → Strategy | 请求链 → Chain of Responsibility
```

| 模式 | Go 实现 | 本项目应用 |
|------|---------|-----------|
| **Factory** | 工厂函数 | `NewOpenAIAdapter()` |
| **Options** | Functional Options | `NewServer(WithPort(8080))` |
| **Singleton** | `sync.Once` | 全局 Logger |
| **Adapter** | 实现统一接口 | Provider 适配器 |
| **Strategy** | interface + 实现 | 路由策略 |
| **Chain** | 中间件链 | Gin middleware |
| **Registry** | map + mutex | ProviderRegistry |

---

## 3️⃣ 开发规范

### 模块结构（标准布局）

```
internal/module/routing/
├── handler.go          # HTTP Handler (Gin)
├── service.go          # 业务逻辑
├── router.go           # 路由策略
├── strategy/           # 策略实现
├── model.go            # 数据模型
├── dto.go              # 请求/响应 DTO
├── errors.go           # 模块错误定义
└── service_test.go     # 单元测试
```

### Handler 层规范

```go
// handler.go
type Handler struct {
    service *Service
}

func NewHandler(service *Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    r.POST("/chat", h.Chat)
    r.POST("/chat/stream", h.ChatStream)
}

func (h *Handler) Chat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    user := c.MustGet("user").(*User)
    resp, err := h.service.Chat(c.Request.Context(), user.ID, &req)
    if err != nil {
        handleError(c, err)
        return
    }

    c.JSON(200, resp)
}
```

### Service 层规范

```go
// service.go
type Service struct {
    repo     Repository
    provider ProviderRegistry
    billing  BillingRecorder
    logger   *zap.Logger
}

func NewService(
    repo Repository,
    provider ProviderRegistry,
    billing BillingRecorder,
    logger *zap.Logger,
) *Service {
    return &Service{
        repo:     repo,
        provider: provider,
        billing:  billing,
        logger:   logger,
    }
}

func (s *Service) Chat(ctx context.Context, userID uuid.UUID, req *ChatRequest) (*ChatResponse, error) {
    // 1. Validate
    // 2. Route to provider
    // 3. Execute request
    // 4. Record usage
    // 5. Return response
}
```

### Repository 层规范

```go
// repository.go
type Repository interface {
    Create(ctx context.Context, model *Model) error
    GetByID(ctx context.Context, id uuid.UUID) (*Model, error)
    List(ctx context.Context, filter Filter) ([]*Model, error)
    Update(ctx context.Context, model *Model) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
    return &repository{db: db}
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Model, error) {
    var model Model
    if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("get by id: %w", err)
    }
    return &model, nil
}
```

### 错误处理规范

```go
// errors.go
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrQuotaExceeded = errors.New("quota exceeded")
)

// 带上下文的错误
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Err     error  `json:"-"`
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// 错误处理中间件
func handleError(c *gin.Context, err error) {
    var appErr *AppError
    if errors.As(err, &appErr) {
        c.JSON(getStatusCode(appErr.Code), appErr)
        return
    }

    if errors.Is(err, ErrNotFound) {
        c.JSON(404, gin.H{"error": "not found"})
        return
    }

    c.JSON(500, gin.H{"error": "internal error"})
}
```

---

## 4️⃣ 测试规范

### 测试流程

```bash
go test ./...                    # 全部测试
go test ./internal/module/auth   # 模块测试
go test -v -run TestChat ./...   # 指定测试
go test -cover ./...             # 覆盖率
go test -race ./...              # 竞态检测
```

### 测试结构

```go
// service_test.go
func TestService_Chat(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        // Arrange
        mockRepo := NewMockRepository(t)
        mockProvider := NewMockProviderRegistry(t)
        svc := NewService(mockRepo, mockProvider, nil)

        mockProvider.EXPECT().
            Get("openai").
            Return(openaiProvider, true)

        // Act
        resp, err := svc.Chat(context.Background(), userID, &ChatRequest{
            Model: "gpt-4o",
        })

        // Assert
        require.NoError(t, err)
        assert.NotNil(t, resp)
    })

    t.Run("provider not found", func(t *testing.T) {
        // ...
    })
}
```

### Mock 生成

```bash
# 使用 mockery 生成 mock
mockery --name=Repository --dir=./internal/module/auth --output=./internal/module/auth/mocks
```

### 测试覆盖目标

| 层级 | 覆盖率 |
|------|--------|
| Service | > 80% |
| Repository | > 70% |
| Handler | > 60% |

---

## 5️⃣ 审查规范

### 评级标准

```
🟢 优秀 - 符合 SOLID，模块清晰，易扩展
🟡 一般 - 基本可用，有改进空间
🔴 问题 - 违反原则，需要重构
```

### 审查要点

```
致命缺陷：
├─ 违反单一职责
├─ 循环依赖
├─ 接口定义在实现方
├─ 忽略错误处理
└─ 缺乏并发安全

改进方向：
├─ 提取接口到使用方
├─ 依赖注入替代全局变量
├─ 增加表驱动测试
└─ 使用 errgroup 处理并发
```

---

## 6️⃣ 文档规范

### 分形文档结构

```
uniedit-server/
├── CLAUDE.md                    # L0: 项目级（本文件）
├── docs/
│   ├── backend-service-design.md  # 架构设计文档
│   └── p0-implementation-tasks.md # P0 任务清单
├── internal/
│   └── module/
│       ├── auth/
│       │   └── README.md        # L1: 模块级
│       └── routing/
│           └── README.md        # L1: 模块级
└── api/
    └── openapi.yaml             # API 文档
```

### 模块 README 模板

```markdown
# Auth Module

## 职责
用户认证与授权管理

## 接口
- `AuthService.Login(provider, code) -> Token`
- `AuthService.Validate(token) -> User`

## 依赖
- `shared/database` - 数据库连接
- `shared/crypto` - 加密工具

## 数据模型
- User
- RefreshToken
- UserAPIKey
```

### 架构可视化

```
图类型选择：
├─ 模块依赖 → graph TB/LR
├─ 请求流程 → sequenceDiagram
├─ 状态机 → stateDiagram
└─ 数据模型 → erDiagram

必须输出架构图：新增模块 | 跨模块交互 | 复杂状态
```

---

## 7️⃣ 检查清单

### 完成任务前必查

**架构**
- [ ] 符合 SOLID 原则
- [ ] 模块充分解耦，无循环依赖
- [ ] 接口定义在使用方

**代码**
- [ ] 错误处理完备
- [ ] 并发安全（适当使用 mutex）
- [ ] 资源释放（defer close）

**测试**
- [ ] 构建通过 `go build ./...`
- [ ] 测试通过 `go test ./...`
- [ ] 代码检查 `golangci-lint run`

**文档**
- [ ] 复杂模块输出架构图
- [ ] 更新模块 README.md

---

## 💡 最后提醒

```
╔════════════════════════════════════════╗
║  写代码前，先问三个问题：               ║
║  1. 是否符合现有架构？                 ║
║  2. 如何最小化耦合？                   ║
║  3. 是否易于扩展测试？                 ║
║                                        ║
║  不确定？先画架构图。                  ║
╚════════════════════════════════════════╝
```
