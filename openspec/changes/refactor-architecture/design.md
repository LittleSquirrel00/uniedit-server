# Design: Architecture Refactoring

## Overview

本文档详细说明架构重构的设计决策和实现方案。

---

## 1. Service 解耦设计

### 1.1 当前依赖关系

```
┌─────────────┐     直接依赖      ┌─────────────┐
│   Payment   │ ───────────────▶ │    Order    │
│   Service   │                   │   Service   │
└─────────────┘                   └─────────────┘
       │
       │ 直接依赖
       ▼
┌─────────────┐
│   Billing   │
│   Service   │
└─────────────┘
```

### 1.2 目标依赖关系

```
┌─────────────┐     依赖接口      ┌─────────────┐
│   Payment   │ ───────────────▶ │    Order    │
│   Service   │                   │  Repository │
└─────────────┘                   └─────────────┘
       │              (interface)
       │ 发布事件
       ▼
┌─────────────┐
│  EventBus   │ ──── PaymentSucceeded ────▶ [Billing Handler]
└─────────────┘                             [Order Handler]
```

### 1.3 接口定义原则

**接口定义在使用方**（依赖倒置原则）：

```go
// payment/deps.go - Payment 模块定义所需接口
package payment

// OrderReader 定义 Payment 模块需要的 Order 读取能力
type OrderReader interface {
    GetOrder(ctx context.Context, id uuid.UUID) (*OrderInfo, error)
    GetOrderByPaymentIntentID(ctx context.Context, id string) (*OrderInfo, error)
}

// OrderInfo 是 Payment 模块需要的 Order 信息（精简视图）
type OrderInfo struct {
    ID       uuid.UUID
    UserID   uuid.UUID
    Type     string
    Status   string
    Total    int64
    Currency string
}
```

---

## 2. 领域事件设计

### 2.1 事件基础设施

```
internal/shared/events/
├── event.go        # 事件接口定义
├── bus.go          # 事件总线实现
├── handler.go      # 处理器接口
└── registry.go     # 处理器注册
```

### 2.2 事件接口

```go
// internal/shared/events/event.go
package events

import (
    "context"
    "time"
    "github.com/google/uuid"
)

// Event 领域事件接口
type Event interface {
    EventID() uuid.UUID
    EventType() string
    OccurredAt() time.Time
    AggregateID() uuid.UUID
    AggregateType() string
}

// BaseEvent 事件基类
type BaseEvent struct {
    ID            uuid.UUID `json:"id"`
    Type          string    `json:"type"`
    Timestamp     time.Time `json:"timestamp"`
    AggID         uuid.UUID `json:"aggregate_id"`
    AggType       string    `json:"aggregate_type"`
}

func (e BaseEvent) EventID() uuid.UUID      { return e.ID }
func (e BaseEvent) EventType() string       { return e.Type }
func (e BaseEvent) OccurredAt() time.Time   { return e.Timestamp }
func (e BaseEvent) AggregateID() uuid.UUID  { return e.AggID }
func (e BaseEvent) AggregateType() string   { return e.AggType }
```

### 2.3 事件总线

```go
// internal/shared/events/bus.go
package events

import (
    "context"
    "sync"
)

// Handler 事件处理器
type Handler interface {
    Handle(ctx context.Context, event Event) error
    Handles() []string // 返回处理的事件类型列表
}

// Bus 事件总线
type Bus struct {
    mu       sync.RWMutex
    handlers map[string][]Handler
    logger   *zap.Logger
}

// NewBus 创建事件总线
func NewBus(logger *zap.Logger) *Bus {
    return &Bus{
        handlers: make(map[string][]Handler),
        logger:   logger,
    }
}

// Register 注册处理器
func (b *Bus) Register(handler Handler) {
    b.mu.Lock()
    defer b.mu.Unlock()
    for _, eventType := range handler.Handles() {
        b.handlers[eventType] = append(b.handlers[eventType], handler)
    }
}

// Publish 发布事件（同步）
func (b *Bus) Publish(ctx context.Context, event Event) error {
    b.mu.RLock()
    handlers := b.handlers[event.EventType()]
    b.mu.RUnlock()

    for _, h := range handlers {
        if err := h.Handle(ctx, event); err != nil {
            b.logger.Error("event handler failed",
                zap.String("event_type", event.EventType()),
                zap.Error(err),
            )
            // 继续处理其他 handler，不中断
        }
    }
    return nil
}
```

### 2.4 核心领域事件

```go
// internal/module/payment/events.go
package payment

import "github.com/uniedit/server/internal/shared/events"

// PaymentSucceeded 支付成功事件
type PaymentSucceeded struct {
    events.BaseEvent
    PaymentID     uuid.UUID `json:"payment_id"`
    OrderID       uuid.UUID `json:"order_id"`
    UserID        uuid.UUID `json:"user_id"`
    Amount        int64     `json:"amount"`
    Currency      string    `json:"currency"`
    Provider      string    `json:"provider"`
    CreditsAmount int64     `json:"credits_amount,omitempty"` // 用于 topup 订单
}

// PaymentFailed 支付失败事件
type PaymentFailed struct {
    events.BaseEvent
    PaymentID      uuid.UUID `json:"payment_id"`
    OrderID        uuid.UUID `json:"order_id"`
    FailureCode    string    `json:"failure_code"`
    FailureMessage string    `json:"failure_message"`
}
```

### 2.5 事件处理器

```go
// internal/module/order/event_handler.go
package order

import (
    "context"
    "github.com/uniedit/server/internal/shared/events"
    "github.com/uniedit/server/internal/module/payment"
)

// EventHandler 处理与订单相关的领域事件
type EventHandler struct {
    repo   Repository
    logger *zap.Logger
}

func NewEventHandler(repo Repository, logger *zap.Logger) *EventHandler {
    return &EventHandler{repo: repo, logger: logger}
}

func (h *EventHandler) Handles() []string {
    return []string{
        "payment.succeeded",
        "payment.failed",
    }
}

func (h *EventHandler) Handle(ctx context.Context, event events.Event) error {
    switch e := event.(type) {
    case *payment.PaymentSucceeded:
        return h.onPaymentSucceeded(ctx, e)
    case *payment.PaymentFailed:
        return h.onPaymentFailed(ctx, e)
    }
    return nil
}

func (h *EventHandler) onPaymentSucceeded(ctx context.Context, e *payment.PaymentSucceeded) error {
    order, err := h.repo.GetOrder(ctx, e.OrderID)
    if err != nil {
        return err
    }

    // 更新订单状态
    now := time.Now()
    order.Status = OrderStatusPaid
    order.PaidAt = &now

    return h.repo.UpdateOrder(ctx, order)
}
```

---

## 3. 模型分离设计

### 3.1 分层结构

```
internal/module/order/
├── domain/                 # 领域层（纯业务逻辑）
│   ├── order.go           # Order 聚合根
│   ├── order_item.go      # OrderItem 实体
│   ├── money.go           # Money 值对象
│   └── status.go          # 状态枚举
├── entity/                 # 持久化层
│   ├── order_entity.go    # OrderEntity (with GORM tags)
│   └── converter.go       # Domain <-> Entity 转换
├── repository.go          # Repository 接口
├── repository_impl.go     # Repository 实现
├── service.go
├── handler.go
└── dto.go
```

### 3.2 领域模型（纯净）

```go
// internal/module/order/domain/order.go
package domain

import (
    "time"
    "github.com/google/uuid"
)

// Order 订单聚合根
type Order struct {
    id            uuid.UUID
    orderNo       string
    userID        uuid.UUID
    orderType     OrderType
    status        OrderStatus
    money         Money
    items         []OrderItem
    createdAt     time.Time
    paidAt        *time.Time
}

// NewOrder 创建订单（工厂方法）
func NewOrder(userID uuid.UUID, orderType OrderType, amount int64, currency string) (*Order, error) {
    if amount <= 0 {
        return nil, ErrInvalidAmount
    }
    return &Order{
        id:        uuid.New(),
        orderNo:   generateOrderNo(),
        userID:    userID,
        orderType: orderType,
        status:    StatusPending,
        money:     NewMoney(amount, currency),
        createdAt: time.Now(),
    }, nil
}

// MarkAsPaid 标记为已支付（领域行为）
func (o *Order) MarkAsPaid() error {
    if o.status != StatusPending {
        return ErrInvalidStatusTransition
    }
    o.status = StatusPaid
    now := time.Now()
    o.paidAt = &now
    return nil
}

// Getters（只读访问）
func (o *Order) ID() uuid.UUID       { return o.id }
func (o *Order) Status() OrderStatus { return o.status }
func (o *Order) IsPending() bool     { return o.status == StatusPending }
```

### 3.3 值对象

```go
// internal/module/order/domain/money.go
package domain

// Money 金额值对象（不可变）
type Money struct {
    amount   int64  // 以分为单位
    currency string
}

func NewMoney(amount int64, currency string) Money {
    return Money{amount: amount, currency: currency}
}

func (m Money) Amount() int64     { return m.amount }
func (m Money) Currency() string  { return m.currency }

func (m Money) Add(other Money) (Money, error) {
    if m.currency != other.currency {
        return Money{}, ErrCurrencyMismatch
    }
    return NewMoney(m.amount + other.amount, m.currency), nil
}

func (m Money) Equals(other Money) bool {
    return m.amount == other.amount && m.currency == other.currency
}
```

### 3.4 持久化实体

```go
// internal/module/order/entity/order_entity.go
package entity

import (
    "time"
    "github.com/google/uuid"
)

// OrderEntity 数据库实体（包含 GORM tags）
type OrderEntity struct {
    ID                    uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    OrderNo               string     `gorm:"uniqueIndex;not null"`
    UserID                uuid.UUID  `gorm:"type:uuid;not null;index"`
    Type                  string     `gorm:"column:type;not null"`
    Status                string     `gorm:"not null;default:pending"`
    Amount                int64      `gorm:"column:total"`
    Currency              string     `gorm:"default:usd"`
    StripePaymentIntentID string
    PaidAt                *time.Time
    CreatedAt             time.Time
    UpdatedAt             time.Time
}

func (OrderEntity) TableName() string {
    return "orders"
}
```

### 3.5 转换器

```go
// internal/module/order/entity/converter.go
package entity

import "github.com/uniedit/server/internal/module/order/domain"

// ToDomain 将持久化实体转换为领域模型
func (e *OrderEntity) ToDomain() *domain.Order {
    return domain.RestoreOrder(
        e.ID,
        e.OrderNo,
        e.UserID,
        domain.OrderType(e.Type),
        domain.OrderStatus(e.Status),
        domain.NewMoney(e.Amount, e.Currency),
        e.CreatedAt,
        e.PaidAt,
    )
}

// FromDomain 将领域模型转换为持久化实体
func FromDomain(o *domain.Order) *OrderEntity {
    return &OrderEntity{
        ID:       o.ID(),
        OrderNo:  o.OrderNo(),
        UserID:   o.UserID(),
        Type:     string(o.OrderType()),
        Status:   string(o.Status()),
        Amount:   o.Money().Amount(),
        Currency: o.Money().Currency(),
        PaidAt:   o.PaidAt(),
    }
}
```

---

## 4. 重构后的 Payment Service

```go
// internal/module/payment/service.go
package payment

type Service struct {
    repo       Repository
    orderRepo  OrderReader      // 接口依赖
    eventBus   *events.Bus      // 事件总线
    registry   *ProviderRegistry
    logger     *zap.Logger
}

func NewService(
    repo Repository,
    orderRepo OrderReader,     // 注入 Repository 接口
    eventBus *events.Bus,      // 注入事件总线
    registry *ProviderRegistry,
    logger *zap.Logger,
) *Service {
    return &Service{
        repo:      repo,
        orderRepo: orderRepo,
        eventBus:  eventBus,
        registry:  registry,
        logger:    logger,
    }
}

func (s *Service) HandlePaymentSucceeded(ctx context.Context, paymentIntentID string) error {
    // 1. 获取订单信息（通过 Repository 接口）
    orderInfo, err := s.orderRepo.GetOrderByPaymentIntentID(ctx, paymentIntentID)
    if err != nil {
        return err
    }

    // 2. 更新支付记录
    payment, _ := s.repo.GetPaymentByPaymentIntentID(ctx, paymentIntentID)
    payment.Status = PaymentStatusSucceeded
    s.repo.UpdatePayment(ctx, payment)

    // 3. 发布领域事件（代替直接调用其他 Service）
    event := &PaymentSucceeded{
        BaseEvent: events.NewBaseEvent("payment.succeeded", payment.ID, "Payment"),
        PaymentID: payment.ID,
        OrderID:   orderInfo.ID,
        UserID:    orderInfo.UserID,
        Amount:    payment.Amount,
        Currency:  payment.Currency,
    }

    return s.eventBus.Publish(ctx, event)
}
```

---

## 5. 依赖组装

```go
// internal/app/app.go
func (a *App) initModules() error {
    // 1. 创建事件总线
    a.eventBus = events.NewBus(a.logger)

    // 2. 创建 Repositories
    orderRepo := order.NewRepository(a.db)
    paymentRepo := payment.NewRepository(a.db)
    billingRepo := billing.NewRepository(a.db)

    // 3. 创建 Services
    orderService := order.NewService(orderRepo, billingRepo, a.logger)
    paymentService := payment.NewService(
        paymentRepo,
        orderRepo,      // 传 Repository，不传 Service
        a.eventBus,
        providerRegistry,
        a.logger,
    )

    // 4. 注册事件处理器
    a.eventBus.Register(order.NewEventHandler(orderRepo, a.logger))
    a.eventBus.Register(billing.NewEventHandler(billingRepo, a.logger))

    return nil
}
```

---

## 6. 迁移策略

### 阶段 1：Service 解耦（低风险）
1. 在 payment 模块定义 `OrderReader` 接口
2. 修改 Service 构造函数，使用接口
3. 在 app.go 中传递 Repository

### 阶段 2：领域事件（中风险）
1. 创建 events 基础设施
2. 定义核心事件类型
3. 逐步替换直接调用为事件发布
4. 添加事件处理器

### 阶段 3：模型分离（高工作量）
1. 先选择一个简单模块（如 order）作为试点
2. 创建 domain/ 和 entity/ 子包
3. 更新 Repository 实现
4. 验证功能正常后，推广到其他模块

---

## 7. 测试策略

### 单元测试
- Service 测试可 mock Repository 接口
- 事件处理器可独立测试
- 领域模型可纯内存测试

### 集成测试
- 验证事件正确发布和处理
- 验证模型转换正确性
- 端到端支付流程测试
