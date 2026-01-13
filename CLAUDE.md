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
| 语言 | Go 1.23+ |
| 框架 | Gin (HTTP) + GORM (ORM) |
| 数据库 | PostgreSQL 16+ |
| 缓存 | Redis 7+ |
| 对象存储 | Cloudflare R2 / MinIO (S3 兼容) |
| 构建工具 | Mage + Wire |
| 支付 | Stripe / Alipay / WeChat Pay |

**项目结构**:

```
uniedit-server/
├── cmd/server/              # 程序入口
├── api/
│   ├── protobuf_spec/       # Proto 接口定义
│   ├── pb/                  # 生成的 Go 代码 (*_pb.go, *_gin.pb.go)
│   └── openapi_spec/        # 生成的 OpenAPI v2 文档
├── internal/
│   ├── app/                 # Wire 依赖注入、应用组装、路由注册
│   ├── adapter/             # 适配层
│   │   ├── inbound/http/    # HTTP Handler (xxxproto/ 使用 Proto 消息)
│   │   └── outbound/        # Postgres/Redis/OAuth/第三方 Provider 适配器
│   ├── domain/              # 领域层 (ai/auth/billing/order/payment/git/collaboration/media/user)
│   ├── infra/               # 基础设施封装 (config/database/cache/httpclient)
│   ├── port/                # 端口定义 (inbound/outbound 接口)
│   ├── model/               # 数据库模型 (GORM)
│   ├── transport/           # 传输层工具 (protohttp 绑定)
│   └── utils/               # 通用工具 (logger/metrics/middleware 等)
├── configs/                 # 配置文件模板
├── migrations/              # 数据库迁移
├── docs/                    # 设计文档
└── openspec/                # OpenSpec 规范
```

**模块依赖关系**:

```
┌─────────────────────────────────────────────────────────────┐
│                      API 层 (Proto-first)                    │
│  api/protobuf_spec/ → api/pb/ → Handler → Domain → Model    │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                         Domain 层                            │
│  ┌─────┐ ┌──────┐ ┌─────────┐ ┌───────┐ ┌─────────────┐     │
│  │ AI  │ │ Auth │ │ Billing │ │ Order │ │ Collaboration│    │
│  └──┬──┘ └──┬───┘ └────┬────┘ └───┬───┘ └──────┬──────┘     │
│     │       │          │          │            │             │
│  ┌──┴──┐ ┌──┴──┐ ┌────┴────┐ ┌───┴───┐ ┌──────┴──────┐     │
│  │Media│ │User │ │ Payment │ │  Git  │ │    ...      │     │
│  └─────┘ └─────┘ └─────────┘ └───────┘ └─────────────┘     │
└─────────────────────────────────────────────────────────────┘

规则：
• Handler 层使用 Proto 消息 (api/pb/)
• Domain 层使用 Proto 消息输入输出
• Model 层仅在数据库边界使用
• 同层模块不互相依赖
```

**构建命令**:

```bash
mage build              # 编译
mage dev                # 构建并运行
mage test               # 测试
mage testCover          # 覆盖率
mage lint               # 代码检查
mage proto              # 生成 Proto 代码
mage wire               # 生成 Wire 依赖注入代码
go build -o bin/server ./cmd/server    # 直接编译
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
1. 定义接口和类型
2. 实现抽象层
3. 编写具体实现
4. 编写测试
```

---

## 2️⃣ 设计原则

### SOLID 原则

```
S - Single Responsibility  → 单一职责：一个模块只做一件事
O - Open/Closed           → 开闭原则：对扩展开放，对修改关闭
L - Liskov Substitution   → 里氏替换：子类可替换父类
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

### API 架构 (Proto-first)

项目采用 Proto-first 的接口定义方式：

```
┌─────────────────────────────────────────────────────────────┐
│  api/protobuf_spec/         定义接口、请求/响应消息          │
│         ↓                                                    │
│  api/pb/                    生成 Go 类型 + Gin Handler 绑定  │
│         ↓                                                    │
│  internal/adapter/inbound/http/xxxproto/   Handler 实现      │
│         ↓                                                    │
│  internal/domain/           业务逻辑 (使用 pb 消息)          │
│         ↓                                                    │
│  internal/model/            数据库模型 (GORM)                │
└─────────────────────────────────────────────────────────────┘
```

| 层级 | 文件位置 | 职责 |
|------|----------|------|
| **接口定义** | `api/protobuf_spec/` | Proto 定义请求/响应消息、Service RPC |
| **生成代码** | `api/pb/` | `*_pb.go` (消息类型) + `*_gin.pb.go` (路由绑定) |
| **Handler** | `internal/adapter/inbound/http/xxxproto/` | 使用 Proto 消息，调用 Domain |
| **Domain** | `internal/domain/` | 业务逻辑，输入输出使用 Proto 消息 |
| **Model** | `internal/model/` | 数据库模型，仅在持久化边界使用 |

### 模块结构（标准布局）

```
internal/domain/auth/
├── domain.go           # 领域服务实现
├── domain_test.go      # 单元测试
├── errors.go           # 模块错误定义
└── mapper.go           # Model <-> Proto 映射

internal/adapter/inbound/http/authproto/
├── handler.go          # HTTP Handler (使用 Proto 消息)
└── handler_test.go     # Handler 测试
```

### Handler 层规范

```go
// handler.go - 使用 Proto 消息
type Handler struct {
    domain *Domain
}

func NewHandler(domain *Domain) *Handler {
    return &Handler{domain: domain}
}

// 实现生成的 Gin Server 接口
func (h *Handler) GetMe(c *gin.Context, req *authv1.GetMeRequest) (*authv1.GetMeResponse, error) {
    userID := middleware.GetUserID(c)
    return h.domain.GetMe(c.Request.Context(), userID)
}

func (h *Handler) Login(c *gin.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
    return h.domain.Login(c.Request.Context(), req)
}
```

### Domain 层规范

```go
// domain.go - 使用 Proto 消息作为输入输出
type Domain struct {
    userDB   UserDBPort
    tokenDB  TokenDBPort
    jwt      JWTPort
    logger   *zap.Logger
}

func NewDomain(
    userDB UserDBPort,
    tokenDB TokenDBPort,
    jwt JWTPort,
    logger *zap.Logger,
) *Domain {
    return &Domain{
        userDB:  userDB,
        tokenDB: tokenDB,
        jwt:     jwt,
        logger:  logger,
    }
}

// 方法使用 Proto 消息
func (d *Domain) GetMe(ctx context.Context, userID uuid.UUID) (*authv1.GetMeResponse, error) {
    user, err := d.userDB.FindByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    return &authv1.GetMeResponse{
        UserId: user.ID.String(),
        Email:  user.Email,
    }, nil
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
mage test                        # 全部测试
mage testCover                   # 带覆盖率
go test ./internal/domain/auth   # 模块测试
go test -v -run TestChat ./...   # 指定测试
go test -race ./...              # 竞态检测
```

### 测试结构

```go
// domain_test.go - Mock 基于接口
type MockUserDB struct {
    mock.Mock
}

func (m *MockUserDB) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.User), args.Error(1)
}

func TestDomain_GetMe(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        // Arrange
        mockUserDB := new(MockUserDB)
        domain := NewDomain(mockUserDB, nil, nil, zap.NewNop())

        userID := uuid.New()
        mockUserDB.On("FindByID", mock.Anything, userID).Return(&model.User{
            ID:    userID,
            Email: "test@example.com",
        }, nil)

        // Act
        resp, err := domain.GetMe(context.Background(), userID)

        // Assert
        require.NoError(t, err)
        assert.Equal(t, userID.String(), resp.GetUserId())
        assert.Equal(t, "test@example.com", resp.GetEmail())
        mockUserDB.AssertExpectations(t)
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
├── README.md                    # 项目介绍、快速开始
├── docs/
│   ├── backend-service-design.md  # 架构设计文档
│   └── design-ai-module.md       # AI 模块设计
├── api/
│   ├── protobuf_spec/           # Proto 接口定义
│   └── openapi_spec/            # OpenAPI 文档
├── internal/
│   └── domain/
│       ├── auth/README.md       # L1: 模块级
│       └── ai/README.md
└── openspec/                    # OpenSpec 规范
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
