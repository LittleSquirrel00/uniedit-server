# UniEdit Server 六边形架构重构设计文档

## 1. 执行摘要

本文档基于 [Hexagonal-Architecture.md](../Hexagonal-Architecture.md) 的设计原则，结合 UniEdit Server 当前架构现状，提出系统性的重构方案。目标是增强模块解耦、提升可测试性、统一架构风格。

### 1.1 当前架构评估

| 维度 | 当前状态 | 目标状态 |
|------|---------|---------|
| **Port/Adapter 覆盖** | 部分模块（Payment, AI） | 所有业务模块 |
| **依赖方向** | 大部分正确，少数反向 | 严格向内依赖 |
| **接口定义位置** | 混合（消费方/实现方） | 统一在消费方 |
| **测试隔离** | 部分支持 Mock | 完全可 Mock |

### 1.2 重构优先级

```
P0 (高优先级): 统一 Port/Adapter 模式
P1 (中优先级): 完善 infra 层
P2 (低优先级): 事件驱动增强
```

---

## 2. 架构对比分析

### 2.1 六边形架构核心概念映射

| Hexagonal 概念 | UniEdit 当前实现 | 重构目标 |
|---------------|-----------------|---------|
| **Domain** | `module/{name}/service.go` | `module/{name}/domain/` |
| **Inbound Port** | `module/{name}/handler.go` | `module/{name}/port/inbound.go` |
| **Outbound Port** | `module/{name}/deps.go` (部分) | `module/{name}/port/outbound.go` |
| **Inbound Adapter** | Gin Handler | `module/{name}/adapter/http/` |
| **Outbound Adapter** | `app/adapters.go` | `module/{name}/adapter/` |
| **Application** | `module/{name}/service.go` | `module/{name}/application/` |

### 2.2 依赖流向对比

**Hexagonal 标准流向：**
```
External System → Inbound Adapter → Inbound Port → Application → Domain
                                                              ↓
External System ← Outbound Adapter ← Outbound Port ←──────────┘
```

**当前实现流向：**
```
HTTP Request → Handler → Service → Repository → Database
                  ↓
            Other Modules (部分通过 Adapter)
```

**重构后流向：**
```
HTTP/gRPC/CLI → Adapter (Inbound) → Port (Inbound) → Application Service
                                                            ↓
                                                        Domain Logic
                                                            ↓
Database/Cache/MQ ← Adapter (Outbound) ← Port (Outbound) ←──┘
```

---

## 3. 目标目录结构

### 3.1 标准模块结构（重构后）

```
internal/module/{name}/
├── domain/                 # 领域层（核心业务逻辑）
│   ├── {entity}.go         # 领域实体
│   ├── value_object.go     # 值对象
│   ├── service.go          # 领域服务
│   └── event.go            # 领域事件
├── port/                   # 端口定义
│   ├── inbound.go          # 入站端口（应用服务接口）
│   └── outbound.go         # 出站端口（依赖接口）
├── adapter/                # 适配器实现
│   ├── inbound/            # 入站适配器
│   │   ├── http/           # HTTP 适配器 (Gin Handler)
│   │   ├── grpc/           # gRPC 适配器
│   │   └── cli/            # CLI 适配器
│   └── outbound/           # 出站适配器
│       ├── postgres/       # 数据库适配器
│       ├── redis/          # 缓存适配器
│       └── http/           # HTTP 客户端适配器
├── application/            # 应用层（用例编排）
│   ├── service.go          # 应用服务实现
│   └── dto.go              # 数据传输对象
├── model.go                # 数据模型（兼容层，渐进迁移）
└── errors.go               # 模块错误定义
```

### 3.2 整体项目结构（重构后）

```
uniedit-server/
├── cmd/server/
│   └── main.go
├── internal/
│   ├── app/                    # 应用组装层
│   │   ├── app.go              # 主应用
│   │   ├── wire.go             # 依赖注入配置
│   │   └── wire_gen.go         # Wire 生成代码
│   ├── module/                 # 业务模块
│   │   ├── auth/
│   │   ├── billing/
│   │   ├── ai/
│   │   ├── payment/
│   │   ├── order/
│   │   ├── git/
│   │   ├── user/
│   │   ├── media/
│   │   └── collaboration/
│   ├── shared/                 # 共享基础设施
│   │   ├── port/               # 共享端口定义
│   │   │   ├── cache.go
│   │   │   ├── database.go
│   │   │   ├── event.go
│   │   │   └── storage.go
│   │   ├── adapter/            # 共享适配器实现
│   │   │   ├── redis/
│   │   │   ├── postgres/
│   │   │   └── r2/
│   │   └── ... (现有 shared 包)
│   └── infra/                  # 基础设施（外部客户端）
│       ├── stripe/
│       ├── openai/
│       └── anthropic/
├── api/                        # API 定义
│   ├── openapi/
│   └── proto/
└── migrations/
```

---

## 4. 重构方案详解

### 4.1 Port 定义规范

#### 4.1.1 Inbound Port（入站端口）

入站端口定义应用服务的公开接口，由外部适配器调用：

```go
// internal/module/billing/port/inbound.go
package port

import (
    "context"
    "github.com/google/uuid"
)

// BillingService defines the inbound port for billing operations.
// This interface is implemented by the application service.
type BillingService interface {
    // Subscription management
    GetSubscription(ctx context.Context, userID uuid.UUID) (*SubscriptionDTO, error)
    CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*SubscriptionDTO, error)
    CancelSubscription(ctx context.Context, userID uuid.UUID) error

    // Credits management
    GetBalance(ctx context.Context, userID uuid.UUID) (*BalanceDTO, error)
    AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error
    DeductCredits(ctx context.Context, userID uuid.UUID, amount int64, reason string) error

    // Usage tracking
    RecordUsage(ctx context.Context, usage UsageInput) error
    GetUsageStats(ctx context.Context, userID uuid.UUID, period Period) (*UsageStatsDTO, error)
}

// DTO definitions
type SubscriptionDTO struct {
    UserID    uuid.UUID
    PlanID    string
    Status    string
    ExpiresAt time.Time
}

type CreateSubscriptionInput struct {
    UserID uuid.UUID
    PlanID string
}
```

#### 4.1.2 Outbound Port（出站端口）

出站端口定义应用服务对外部系统的依赖，由适配器实现：

```go
// internal/module/billing/port/outbound.go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/uniedit/server/internal/module/billing/domain"
)

// SubscriptionRepository defines the outbound port for subscription persistence.
type SubscriptionRepository interface {
    Save(ctx context.Context, sub *domain.Subscription) error
    FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
    Update(ctx context.Context, sub *domain.Subscription) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// BalanceRepository defines the outbound port for balance persistence.
type BalanceRepository interface {
    Get(ctx context.Context, userID uuid.UUID) (*domain.Balance, error)
    Save(ctx context.Context, balance *domain.Balance) error
    IncrementCredits(ctx context.Context, userID uuid.UUID, amount int64) error
    DecrementCredits(ctx context.Context, userID uuid.UUID, amount int64) error
}

// UsageRepository defines the outbound port for usage persistence.
type UsageRepository interface {
    Record(ctx context.Context, usage *domain.Usage) error
    GetByUserAndPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.Usage, error)
    Aggregate(ctx context.Context, userID uuid.UUID, period Period) (*domain.UsageAggregate, error)
}

// PaymentGateway defines the outbound port for payment processing.
// This abstracts external payment providers (Stripe, Alipay, etc.)
type PaymentGateway interface {
    CreateCustomer(ctx context.Context, userID uuid.UUID, email string) (string, error)
    CreateSubscription(ctx context.Context, customerID, priceID string) (string, error)
    CancelSubscription(ctx context.Context, subscriptionID string) error
}

// EventPublisher defines the outbound port for publishing domain events.
type EventPublisher interface {
    Publish(ctx context.Context, event domain.Event) error
}

// CachePort defines the outbound port for caching.
type CachePort interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

### 4.2 Adapter 实现规范

#### 4.2.1 Inbound Adapter（HTTP Handler）

```go
// internal/module/billing/adapter/inbound/http/handler.go
package http

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/uniedit/server/internal/module/billing/port"
)

// Handler implements the HTTP adapter for billing operations.
type Handler struct {
    service port.BillingService // Depends on port, not concrete implementation
}

// NewHandler creates a new HTTP handler.
func NewHandler(service port.BillingService) *Handler {
    return &Handler{service: service}
}

// RegisterRoutes registers all HTTP routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    billing := r.Group("/billing")
    {
        billing.GET("/subscription", h.GetSubscription)
        billing.POST("/subscription", h.CreateSubscription)
        billing.DELETE("/subscription", h.CancelSubscription)
        billing.GET("/balance", h.GetBalance)
        billing.GET("/usage", h.GetUsageStats)
    }
}

// GetSubscription handles GET /billing/subscription
func (h *Handler) GetSubscription(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)

    sub, err := h.service.GetSubscription(c.Request.Context(), userID)
    if err != nil {
        handleError(c, err)
        return
    }

    c.JSON(200, toSubscriptionResponse(sub))
}
```

#### 4.2.2 Outbound Adapter（Database）

```go
// internal/module/billing/adapter/outbound/postgres/subscription_repo.go
package postgres

import (
    "context"
    "github.com/google/uuid"
    "github.com/uniedit/server/internal/module/billing/domain"
    "github.com/uniedit/server/internal/module/billing/port"
    "gorm.io/gorm"
)

// subscriptionRepo implements port.SubscriptionRepository.
type subscriptionRepo struct {
    db *gorm.DB
}

// NewSubscriptionRepository creates a new subscription repository.
func NewSubscriptionRepository(db *gorm.DB) port.SubscriptionRepository {
    return &subscriptionRepo{db: db}
}

// Save persists a subscription.
func (r *subscriptionRepo) Save(ctx context.Context, sub *domain.Subscription) error {
    entity := toSubscriptionEntity(sub)
    return r.db.WithContext(ctx).Create(entity).Error
}

// FindByUserID finds a subscription by user ID.
func (r *subscriptionRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
    var entity SubscriptionEntity
    if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&entity).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return toDomainSubscription(&entity), nil
}

// Compile-time check
var _ port.SubscriptionRepository = (*subscriptionRepo)(nil)
```

### 4.3 Application Service 实现

```go
// internal/module/billing/application/service.go
package application

import (
    "context"
    "github.com/google/uuid"
    "github.com/uniedit/server/internal/module/billing/domain"
    "github.com/uniedit/server/internal/module/billing/port"
)

// Service implements port.BillingService.
type Service struct {
    subRepo      port.SubscriptionRepository
    balanceRepo  port.BalanceRepository
    usageRepo    port.UsageRepository
    paymentGw    port.PaymentGateway
    eventPub     port.EventPublisher
    cache        port.CachePort
}

// NewService creates a new billing application service.
func NewService(
    subRepo port.SubscriptionRepository,
    balanceRepo port.BalanceRepository,
    usageRepo port.UsageRepository,
    paymentGw port.PaymentGateway,
    eventPub port.EventPublisher,
    cache port.CachePort,
) port.BillingService {
    return &Service{
        subRepo:     subRepo,
        balanceRepo: balanceRepo,
        usageRepo:   usageRepo,
        paymentGw:   paymentGw,
        eventPub:    eventPub,
        cache:       cache,
    }
}

// GetSubscription retrieves a user's subscription.
func (s *Service) GetSubscription(ctx context.Context, userID uuid.UUID) (*port.SubscriptionDTO, error) {
    // 1. Try cache first
    // 2. Query repository
    // 3. Transform to DTO
    sub, err := s.subRepo.FindByUserID(ctx, userID)
    if err != nil {
        return nil, err
    }
    if sub == nil {
        return nil, ErrSubscriptionNotFound
    }
    return toSubscriptionDTO(sub), nil
}

// AddCredits adds credits to a user's balance.
func (s *Service) AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error {
    // 1. Get or create balance
    balance, err := s.balanceRepo.Get(ctx, userID)
    if err != nil {
        return err
    }

    // 2. Domain logic
    if err := balance.AddCredits(amount, source); err != nil {
        return err
    }

    // 3. Persist
    if err := s.balanceRepo.Save(ctx, balance); err != nil {
        return err
    }

    // 4. Publish event
    event := domain.NewCreditsAddedEvent(userID, amount, source)
    return s.eventPub.Publish(ctx, event)
}

// Compile-time check
var _ port.BillingService = (*Service)(nil)
```

### 4.4 Domain Layer 实现

```go
// internal/module/billing/domain/subscription.go
package domain

import (
    "errors"
    "time"
    "github.com/google/uuid"
)

// Subscription represents the subscription aggregate root.
type Subscription struct {
    id               uuid.UUID
    userID           uuid.UUID
    planID           string
    status           SubscriptionStatus
    stripeCustomerID string
    stripeSubID      string
    currentPeriodEnd time.Time
    createdAt        time.Time
    updatedAt        time.Time
}

// SubscriptionStatus defines possible subscription states.
type SubscriptionStatus string

const (
    StatusActive   SubscriptionStatus = "active"
    StatusCanceled SubscriptionStatus = "canceled"
    StatusPastDue  SubscriptionStatus = "past_due"
    StatusTrialing SubscriptionStatus = "trialing"
)

// NewSubscription creates a new subscription.
func NewSubscription(userID uuid.UUID, planID string) *Subscription {
    return &Subscription{
        id:        uuid.New(),
        userID:    userID,
        planID:    planID,
        status:    StatusActive,
        createdAt: time.Now(),
        updatedAt: time.Now(),
    }
}

// Cancel cancels the subscription.
func (s *Subscription) Cancel() error {
    if s.status == StatusCanceled {
        return errors.New("subscription already canceled")
    }
    s.status = StatusCanceled
    s.updatedAt = time.Now()
    return nil
}

// IsActive checks if the subscription is active.
func (s *Subscription) IsActive() bool {
    return s.status == StatusActive || s.status == StatusTrialing
}

// Getters
func (s *Subscription) ID() uuid.UUID               { return s.id }
func (s *Subscription) UserID() uuid.UUID           { return s.userID }
func (s *Subscription) PlanID() string              { return s.planID }
func (s *Subscription) Status() SubscriptionStatus  { return s.status }
func (s *Subscription) StripeCustomerID() string    { return s.stripeCustomerID }
```

---

## 5. 跨模块依赖处理

### 5.1 当前问题

Payment 模块需要读取 Order 和 Billing 信息，当前通过 `app/adapters.go` 解决，这是正确的做法。

### 5.2 统一跨模块端口规范

```go
// internal/module/payment/port/outbound.go
package port

// OrderReader defines the outbound port for reading order information.
// This interface is defined in the payment module (consumer) and
// implemented by an adapter in the app package.
type OrderReader interface {
    GetOrder(ctx context.Context, id uuid.UUID) (*OrderInfo, error)
    GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*OrderInfo, error)
    UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error
    SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error
}

// BillingReader defines the outbound port for reading billing information.
type BillingReader interface {
    GetSubscription(ctx context.Context, userID uuid.UUID) (*SubscriptionInfo, error)
    AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error
}

// EventPublisher defines the outbound port for publishing domain events.
type EventPublisher interface {
    Publish(event interface{})
}
```

### 5.3 适配器位置规范

| 场景 | 适配器位置 | 原因 |
|------|-----------|------|
| 模块内部依赖 | `module/{name}/adapter/outbound/` | 模块自包含 |
| 跨模块依赖 | `internal/app/adapters.go` | 避免循环依赖 |
| 共享基础设施 | `internal/shared/adapter/` | 多模块复用 |

---

## 6. 渐进式重构计划

### 6.1 Phase 1: 端口定义统一（低风险）

**目标**: 为所有模块定义标准的 Port 接口

**步骤**:
1. 在每个模块创建 `port/` 目录
2. 提取现有接口到 `port/inbound.go` 和 `port/outbound.go`
3. 确保接口定义在消费方

**影响范围**: 仅添加文件，不修改现有代码

```bash
# 创建目录结构
mkdir -p internal/module/{auth,billing,order,payment,git,user,media,collaboration}/port
```

### 6.2 Phase 2: Adapter 重组（中等风险）

**目标**: 将 Handler 和 Repository 重组为标准 Adapter 结构

**步骤**:
1. 创建 `adapter/inbound/http/` 目录
2. 将现有 `handler.go` 移动到新位置
3. 创建 `adapter/outbound/postgres/` 目录
4. 将现有 `repository.go` 移动到新位置
5. 更新 import 路径

**示例迁移**:
```
# Before
internal/module/billing/handler.go
internal/module/billing/repository.go

# After
internal/module/billing/adapter/inbound/http/handler.go
internal/module/billing/adapter/outbound/postgres/repository.go
```

### 6.3 Phase 3: Domain 提取（高风险）

**目标**: 将业务逻辑提取到纯净的 Domain 层

**步骤**:
1. 识别现有 Service 中的领域逻辑
2. 创建 `domain/` 目录和领域实体
3. 将业务规则移入领域实体
4. Service 层保留用例编排

**注意**: 此阶段需要逐模块进行，每个模块完成后进行充分测试

### 6.4 Phase 4: Application 层分离

**目标**: 清晰分离 Application Service 和 Domain Service

**步骤**:
1. 创建 `application/` 目录
2. 将现有 `service.go` 重命名/移动
3. 确保 Application Service 只做编排
4. 领域逻辑下沉到 Domain 层

---

## 7. 模块重构详细计划

### 7.1 Billing 模块重构

**当前结构**:
```
internal/module/billing/
├── handler.go
├── service.go
├── repository.go
├── model.go
├── dto.go
├── errors.go
├── quota/
└── usage/
```

**目标结构**:
```
internal/module/billing/
├── domain/
│   ├── subscription.go
│   ├── balance.go
│   ├── usage.go
│   ├── plan.go
│   └── event.go
├── port/
│   ├── inbound.go
│   └── outbound.go
├── adapter/
│   ├── inbound/
│   │   └── http/
│   │       ├── handler.go
│   │       └── dto.go
│   └── outbound/
│       ├── postgres/
│       │   ├── subscription_repo.go
│       │   ├── balance_repo.go
│       │   └── usage_repo.go
│       └── redis/
│           └── cache.go
├── application/
│   ├── service.go
│   └── dto.go
├── quota/                  # 保留现有子模块
└── usage/                  # 保留现有子模块
```

### 7.2 AI 模块优化

AI 模块已经有良好的 Adapter 模式，主要优化：

1. 提取 `port/` 定义
2. 统一 Adapter 接口命名
3. 增强 Domain 层

**当前优势**:
- 已有 `adapter/` 目录
- 已实现多提供商适配器
- 使用 Google Wire

**优化点**:
```
internal/module/ai/
├── port/                   # 新增
│   ├── inbound.go          # LLM Service 接口
│   └── outbound.go         # Provider, Cache, Billing 接口
├── domain/                 # 增强
│   ├── model.go            # 模型定义
│   ├── provider.go         # 提供商领域逻辑
│   └── routing.go          # 路由领域逻辑
└── ... (现有结构保持)
```

---

## 8. 测试策略

### 8.1 测试金字塔

```
         /\
        /  \
       / E2E\        <- 端到端测试（少量）
      /------\
     /Integration\   <- 集成测试（适量）
    /------------\
   /  Unit Tests  \  <- 单元测试（大量）
  /----------------\
```

### 8.2 分层测试策略

| 层级 | 测试类型 | Mock 范围 | 覆盖率目标 |
|------|---------|----------|-----------|
| Domain | 单元测试 | 无需 Mock | > 90% |
| Application | 单元测试 | Mock Outbound Ports | > 80% |
| Adapter | 集成测试 | 使用测试容器 | > 70% |
| Handler | 单元测试 | Mock Application Service | > 60% |

### 8.3 Mock 生成

```bash
# 使用 mockery 生成 mock
mockery --dir=internal/module/billing/port --name=SubscriptionRepository --output=internal/module/billing/mocks

# 或使用 gomock
mockgen -source=internal/module/billing/port/outbound.go -destination=internal/module/billing/mocks/outbound_mock.go
```

### 8.4 测试示例

```go
// internal/module/billing/application/service_test.go
func TestService_AddCredits(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        // Arrange
        mockBalanceRepo := mocks.NewMockBalanceRepository(t)
        mockEventPub := mocks.NewMockEventPublisher(t)

        svc := NewService(nil, mockBalanceRepo, nil, nil, mockEventPub, nil)

        userID := uuid.New()
        balance := domain.NewBalance(userID)

        mockBalanceRepo.EXPECT().
            Get(mock.Anything, userID).
            Return(balance, nil)

        mockBalanceRepo.EXPECT().
            Save(mock.Anything, mock.Anything).
            Return(nil)

        mockEventPub.EXPECT().
            Publish(mock.Anything, mock.Anything).
            Return(nil)

        // Act
        err := svc.AddCredits(context.Background(), userID, 1000, "purchase")

        // Assert
        require.NoError(t, err)
    })
}
```

---

## 9. 依赖注入增强

### 9.1 使用 Google Wire

```go
// internal/module/billing/wire.go
//go:build wireinject

package billing

import (
    "github.com/google/wire"
    "github.com/uniedit/server/internal/module/billing/adapter/inbound/http"
    "github.com/uniedit/server/internal/module/billing/adapter/outbound/postgres"
    redisadapter "github.com/uniedit/server/internal/module/billing/adapter/outbound/redis"
    "github.com/uniedit/server/internal/module/billing/application"
    "github.com/uniedit/server/internal/module/billing/port"
    "gorm.io/gorm"
    "github.com/redis/go-redis/v9"
)

var ProviderSet = wire.NewSet(
    // Outbound adapters
    postgres.NewSubscriptionRepository,
    postgres.NewBalanceRepository,
    postgres.NewUsageRepository,
    redisadapter.NewCacheAdapter,

    // Application service
    application.NewService,
    wire.Bind(new(port.BillingService), new(*application.Service)),

    // Inbound adapters
    http.NewHandler,
)

func InitializeBillingModule(db *gorm.DB, rdb redis.UniversalClient, eventPub port.EventPublisher, paymentGw port.PaymentGateway) (*http.Handler, error) {
    wire.Build(ProviderSet)
    return nil, nil
}
```

### 9.2 App 层组装

```go
// internal/app/app.go
func (a *App) initBillingModule() error {
    // Create adapters for cross-module dependencies
    eventPub := newEventBusAdapter(a.eventBus)
    paymentGw := newStripeAdapter(a.config.Stripe)

    // Initialize module with Wire
    handler, err := billing.InitializeBillingModule(a.db, a.redis, eventPub, paymentGw)
    if err != nil {
        return err
    }

    a.billingHandler = handler
    return nil
}
```

---

## 10. 架构决策记录

### ADR-001: Port 定义位置

**决策**: 接口定义在消费方模块的 `port/outbound.go`

**原因**:
1. 遵循依赖倒置原则
2. 避免循环依赖
3. 消费方定义所需接口

### ADR-002: 跨模块适配器位置

**决策**: 跨模块适配器统一放在 `internal/app/adapters.go`

**原因**:
1. 集中管理跨模块依赖
2. 便于追踪模块间关系
3. 简化依赖注入

### ADR-003: 渐进式迁移策略

**决策**: 分四个阶段渐进式重构

**原因**:
1. 降低风险
2. 持续交付
3. 便于回滚

---

## 11. 检查清单

### 重构前检查

- [ ] 所有测试通过
- [ ] 无编译错误
- [ ] 已备份当前状态

### 每阶段检查

- [ ] 新代码符合目录结构规范
- [ ] 接口定义在正确位置
- [ ] 依赖方向正确（向内依赖）
- [ ] Mock 可正常生成
- [ ] 单元测试通过
- [ ] 集成测试通过

### 重构后检查

- [ ] 所有模块结构统一
- [ ] 无循环依赖
- [ ] 测试覆盖率达标
- [ ] 文档已更新

---

## 12. 参考资料

1. [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
2. [Ports and Adapters Pattern](https://herbertograca.com/2017/09/14/ports-adapters-architecture/)
3. [Clean Architecture in Go](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
4. [Google Wire](https://github.com/google/wire)
5. [GORM Documentation](https://gorm.io/docs/)
