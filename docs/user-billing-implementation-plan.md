# 用户管理、产品套餐、订单、支付模块实施计划

## 目标

实现完整的用户管理、订阅计费、订单管理和 Stripe 支付集成功能。

## 需求确认

| 需求 | 选择 |
|------|------|
| 注册方式 | OAuth + 邮箱密码注册 |
| 支付范围 | 订阅 + 余额充值 |
| 用户管理 | 用户自助注销 + 管理员启用/禁用 |
| 订单系统 | 完整订单流程（创建、支付、退款、发票） |

---

## 一、模块架构

```
┌─────────────────────────────────────────────────────────────┐
│                        API Gateway                           │
│              (Gin Router + Middleware + Auth)                │
└─────────────────────────────┬───────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│     User     │      │    Order     │◄─────│   Payment    │
│   Service    │      │   Service    │      │   Service    │
│              │      │              │      │              │
│ • 邮箱注册   │      │ • 订单创建   │      │ • Stripe集成 │
│ • 用户状态   │      │ • 订单状态   │      │ • Webhook    │
│ • 管理员操作 │      │ • 发票生成   │      │ • 支付回调   │
└──────┬───────┘      └──────┬───────┘      └──────┬───────┘
       │                     │                     │
       └─────────────────────┼─────────────────────┘
                             │
                      ┌──────▼──────┐
                      │  Billing    │
                      │  Service    │
                      │             │
                      │ • 订阅管理  │
                      │ • 配额管理  │
                      │ • 用量统计  │
                      └─────────────┘
```

### 模块职责划分

| 模块 | 职责 | 依赖 |
|------|------|------|
| **user** | 用户管理增强：邮箱注册、状态管理、管理员操作 | auth (复用 JWT/Crypto) |
| **billing** | 订阅管理、配额管理、用量记录、套餐定义 | user, redis |
| **order** | 订单生命周期、发票管理 | user, billing |
| **payment** | Stripe 集成、Webhook 处理、支付抽象 | order, billing |

---

## 二、目录结构

```
internal/module/
├── user/                        # 用户管理模块（增强 auth）
│   ├── model.go                 # 用户模型扩展
│   ├── repository.go            # 数据访问接口和实现
│   ├── service.go               # 业务逻辑
│   ├── handler.go               # HTTP Handler
│   ├── admin_handler.go         # 管理员接口
│   ├── verification.go          # 邮箱验证服务
│   ├── dto.go                   # 请求/响应 DTO
│   └── errors.go                # 模块错误定义
│
├── billing/                     # 计费模块
│   ├── model.go                 # Subscription, Plan, UsageRecord
│   ├── repository.go            # 数据访问
│   ├── service.go               # 业务逻辑
│   ├── handler.go               # HTTP Handler
│   ├── quota/                   # 配额管理子模块
│   │   ├── manager.go           # QuotaManager
│   │   └── checker.go           # QuotaChecker 中间件
│   ├── usage/                   # 用量记录子模块
│   │   ├── recorder.go          # UsageRecorder 接口
│   │   └── aggregator.go        # 统计聚合
│   ├── dto.go
│   └── errors.go
│
├── order/                       # 订单模块
│   ├── model.go                 # Order, OrderItem, Invoice
│   ├── repository.go
│   ├── service.go
│   ├── handler.go
│   ├── state_machine.go         # 订单状态机
│   ├── invoice/                 # 发票子模块
│   │   ├── generator.go
│   │   └── template.go
│   ├── dto.go
│   └── errors.go
│
└── payment/                     # 支付模块
    ├── model.go                 # Payment, PaymentMethod
    ├── repository.go
    ├── service.go
    ├── handler.go               # 用户支付接口
    ├── webhook_handler.go       # Webhook 处理
    ├── provider/                # 支付提供商抽象
    │   ├── provider.go          # PaymentProvider 接口
    │   ├── stripe.go            # Stripe 实现
    │   └── registry.go          # 提供商注册表
    ├── dto.go
    └── errors.go
```

---

## 三、核心数据模型

### 3.1 用户模型扩展

```go
// internal/module/user/model.go

// UserStatus 用户状态
type UserStatus string

const (
    UserStatusActive    UserStatus = "active"
    UserStatusPending   UserStatus = "pending"    // 等待邮箱验证
    UserStatusSuspended UserStatus = "suspended"  // 管理员暂停
    UserStatusDeleted   UserStatus = "deleted"    // 自助注销（软删除）
)

// User 扩展用户模型
type User struct {
    ID            uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Email         string        `json:"email" gorm:"uniqueIndex;not null"`
    Name          string        `json:"name" gorm:"not null"`
    AvatarURL     string        `json:"avatar_url,omitempty"`

    // 认证方式
    OAuthProvider *string       `json:"oauth_provider,omitempty"`  // github, google, 可为空
    OAuthID       *string       `json:"-"`                         // OAuth ID
    PasswordHash  *string       `json:"-"`                         // 邮箱注册用户的密码

    // 状态管理
    Status        UserStatus    `json:"status" gorm:"default:active"`
    EmailVerified bool          `json:"email_verified" gorm:"default:false"`

    // 管理员标记
    IsAdmin       bool          `json:"is_admin" gorm:"default:false"`

    // 时间戳
    CreatedAt     time.Time     `json:"created_at"`
    UpdatedAt     time.Time     `json:"updated_at"`
    DeletedAt     *time.Time    `json:"-" gorm:"index"`  // 软删除
    SuspendedAt   *time.Time    `json:"suspended_at,omitempty"`
    SuspendReason *string       `json:"suspend_reason,omitempty"`
}

// EmailVerification 邮箱验证令牌
type EmailVerification struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
    Token     string    `gorm:"not null;uniqueIndex"`
    Purpose   string    `gorm:"not null"` // registration, password_reset
    ExpiresAt time.Time `gorm:"not null"`
    UsedAt    *time.Time
    CreatedAt time.Time
}
```

### 3.2 订阅和套餐模型

```go
// internal/module/billing/model.go

// PlanType 套餐类型
type PlanType string

const (
    PlanTypeFree       PlanType = "free"
    PlanTypePro        PlanType = "pro"
    PlanTypeTeam       PlanType = "team"
    PlanTypeEnterprise PlanType = "enterprise"
)

// BillingCycle 计费周期
type BillingCycle string

const (
    BillingCycleMonthly BillingCycle = "monthly"
    BillingCycleYearly  BillingCycle = "yearly"
)

// Plan 套餐定义
type Plan struct {
    ID              string          `json:"id" gorm:"primaryKey"`           // free, pro_monthly, pro_yearly
    Type            PlanType        `json:"type" gorm:"not null"`
    Name            string          `json:"name" gorm:"not null"`
    Description     string          `json:"description"`
    BillingCycle    BillingCycle    `json:"billing_cycle"`                  // free 套餐为空

    // 定价
    PriceUSD        int64           `json:"price_usd"`                      // 单位：美分
    StripePriceID   string          `json:"-"`                              // Stripe Price ID

    // 配额限制
    MonthlyTokens   int64           `json:"monthly_tokens"`                 // -1 表示无限
    DailyRequests   int             `json:"daily_requests"`                 // -1 表示无限
    MaxAPIKeys      int             `json:"max_api_keys"`

    // 功能特性
    Features        pq.StringArray  `json:"features" gorm:"type:text[]"`

    // 状态
    Active          bool            `json:"active" gorm:"default:true"`
    DisplayOrder    int             `json:"display_order" gorm:"default:0"`

    CreatedAt       time.Time       `json:"created_at"`
    UpdatedAt       time.Time       `json:"updated_at"`
}

// SubscriptionStatus 订阅状态
type SubscriptionStatus string

const (
    SubscriptionStatusTrialing    SubscriptionStatus = "trialing"
    SubscriptionStatusActive      SubscriptionStatus = "active"
    SubscriptionStatusPastDue     SubscriptionStatus = "past_due"
    SubscriptionStatusCanceled    SubscriptionStatus = "canceled"
    SubscriptionStatusIncomplete  SubscriptionStatus = "incomplete"
)

// Subscription 用户订阅
type Subscription struct {
    ID                   uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    UserID               uuid.UUID          `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
    PlanID               string             `json:"plan_id" gorm:"not null"`
    Status               SubscriptionStatus `json:"status" gorm:"not null;default:active"`

    // Stripe 信息
    StripeCustomerID     string             `json:"-"`
    StripeSubscriptionID string             `json:"-"`

    // 计费周期
    CurrentPeriodStart   time.Time          `json:"current_period_start"`
    CurrentPeriodEnd     time.Time          `json:"current_period_end"`

    // 取消信息
    CancelAtPeriodEnd    bool               `json:"cancel_at_period_end" gorm:"default:false"`
    CanceledAt           *time.Time         `json:"canceled_at,omitempty"`

    // 余额（充值模式）
    CreditsBalance       int64              `json:"credits_balance" gorm:"default:0"` // 单位：美分

    CreatedAt            time.Time          `json:"created_at"`
    UpdatedAt            time.Time          `json:"updated_at"`

    // Relations
    Plan                 *Plan              `json:"plan,omitempty" gorm:"foreignKey:PlanID"`
}

// UsageRecord 用量记录（TimescaleDB 超表）
type UsageRecord struct {
    ID           int64     `gorm:"primaryKey;autoIncrement"`
    UserID       uuid.UUID `gorm:"type:uuid;not null"`
    Timestamp    time.Time `gorm:"not null"`                   // 分区键

    // 请求信息
    RequestID    string    `gorm:"not null"`
    TaskType     string    `gorm:"not null"`                   // chat, image, video, embedding
    ProviderID   uuid.UUID `gorm:"type:uuid;not null"`
    ModelID      string    `gorm:"not null"`

    // 用量
    InputTokens  int       `gorm:"not null;default:0"`
    OutputTokens int       `gorm:"not null;default:0"`
    TotalTokens  int       `gorm:"not null;default:0"`

    // 成本
    CostUSD      float64   `gorm:"type:decimal(10,6);not null"`

    // 性能
    LatencyMs    int       `gorm:"not null"`
    Success      bool      `gorm:"not null"`
}
```

### 3.3 订单模型

```go
// internal/module/order/model.go

// OrderStatus 订单状态
type OrderStatus string

const (
    OrderStatusPending   OrderStatus = "pending"
    OrderStatusPaid      OrderStatus = "paid"
    OrderStatusCanceled  OrderStatus = "canceled"
    OrderStatusRefunded  OrderStatus = "refunded"
    OrderStatusFailed    OrderStatus = "failed"
)

// OrderType 订单类型
type OrderType string

const (
    OrderTypeSubscription OrderType = "subscription"  // 订阅
    OrderTypeTopup        OrderType = "topup"         // 充值
    OrderTypeUpgrade      OrderType = "upgrade"       // 升级
)

// Order 订单
type Order struct {
    ID              uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    OrderNo         string       `json:"order_no" gorm:"uniqueIndex;not null"` // ORD-20260106-XXXXX
    UserID          uuid.UUID    `json:"user_id" gorm:"type:uuid;not null;index"`

    // 订单信息
    Type            OrderType    `json:"type" gorm:"not null"`
    Status          OrderStatus  `json:"status" gorm:"not null;default:pending"`

    // 金额
    Subtotal        int64        `json:"subtotal"`       // 原价（美分）
    Discount        int64        `json:"discount"`       // 折扣
    Tax             int64        `json:"tax"`            // 税费
    Total           int64        `json:"total"`          // 总价
    Currency        string       `json:"currency" gorm:"default:usd"`

    // 关联
    PlanID          *string      `json:"plan_id,omitempty"`              // 订阅订单
    CreditsAmount   int64        `json:"credits_amount,omitempty"`       // 充值金额

    // Stripe
    StripePaymentIntentID string `json:"-"`
    StripeInvoiceID       string `json:"-"`

    // 时间戳
    PaidAt          *time.Time   `json:"paid_at,omitempty"`
    CanceledAt      *time.Time   `json:"canceled_at,omitempty"`
    RefundedAt      *time.Time   `json:"refunded_at,omitempty"`
    ExpiresAt       *time.Time   `json:"expires_at,omitempty"`           // 支付过期时间
    CreatedAt       time.Time    `json:"created_at"`
    UpdatedAt       time.Time    `json:"updated_at"`

    // Relations
    Items           []OrderItem  `json:"items,omitempty" gorm:"foreignKey:OrderID"`
    Payments        []Payment    `json:"payments,omitempty" gorm:"foreignKey:OrderID"`
}

// OrderItem 订单项
type OrderItem struct {
    ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    OrderID     uuid.UUID `json:"order_id" gorm:"type:uuid;not null"`
    Description string    `json:"description" gorm:"not null"`
    Quantity    int       `json:"quantity" gorm:"default:1"`
    UnitPrice   int64     `json:"unit_price"`
    Amount      int64     `json:"amount"`
}

// Invoice 发票
type Invoice struct {
    ID              uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    InvoiceNo       string     `json:"invoice_no" gorm:"uniqueIndex;not null"` // INV-20260106-XXXXX
    OrderID         uuid.UUID  `json:"order_id" gorm:"type:uuid;not null"`
    UserID          uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`

    // 发票信息
    Amount          int64      `json:"amount"`
    Currency        string     `json:"currency"`
    Status          string     `json:"status"` // draft, finalized, paid, void

    // PDF 存储
    PDFURL          string     `json:"pdf_url,omitempty"`
    StripeInvoiceID string     `json:"-"`

    // 时间
    IssuedAt        time.Time  `json:"issued_at"`
    DueAt           time.Time  `json:"due_at"`
    PaidAt          *time.Time `json:"paid_at,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
}
```

### 3.4 支付模型

```go
// internal/module/payment/model.go

// PaymentStatus 支付状态
type PaymentStatus string

const (
    PaymentStatusPending    PaymentStatus = "pending"
    PaymentStatusProcessing PaymentStatus = "processing"
    PaymentStatusSucceeded  PaymentStatus = "succeeded"
    PaymentStatusFailed     PaymentStatus = "failed"
    PaymentStatusCanceled   PaymentStatus = "canceled"
    PaymentStatusRefunded   PaymentStatus = "refunded"
)

// PaymentMethod 支付方式
type PaymentMethod string

const (
    PaymentMethodCard   PaymentMethod = "card"
    PaymentMethodAlipay PaymentMethod = "alipay"
    PaymentMethodWechat PaymentMethod = "wechat"
)

// Payment 支付记录
type Payment struct {
    ID                    uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    OrderID               uuid.UUID     `json:"order_id" gorm:"type:uuid;not null;index"`
    UserID                uuid.UUID     `json:"user_id" gorm:"type:uuid;not null;index"`

    // 支付信息
    Amount                int64         `json:"amount"`
    Currency              string        `json:"currency"`
    Method                PaymentMethod `json:"method"`
    Status                PaymentStatus `json:"status" gorm:"not null;default:pending"`

    // 提供商信息
    Provider              string        `json:"provider"` // stripe
    StripePaymentIntentID string        `json:"-"`
    StripeChargeID        string        `json:"-"`

    // 失败信息
    FailureCode           *string       `json:"failure_code,omitempty"`
    FailureMessage        *string       `json:"failure_message,omitempty"`

    // 退款
    RefundedAmount        int64         `json:"refunded_amount" gorm:"default:0"`

    // 时间戳
    SucceededAt           *time.Time    `json:"succeeded_at,omitempty"`
    FailedAt              *time.Time    `json:"failed_at,omitempty"`
    CreatedAt             time.Time     `json:"created_at"`
    UpdatedAt             time.Time     `json:"updated_at"`
}

// StripeWebhookEvent Webhook 事件记录
type StripeWebhookEvent struct {
    ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    EventID     string    `gorm:"uniqueIndex;not null"` // Stripe event ID
    Type        string    `gorm:"not null"`
    Data        string    `gorm:"type:jsonb"`
    Processed   bool      `gorm:"default:false"`
    ProcessedAt *time.Time
    Error       *string
    CreatedAt   time.Time
}
```

---

## 四、接口设计

### 4.1 用户服务接口

```go
// internal/module/user/service.go

// UserService 用户服务接口
type UserService interface {
    // 邮箱注册
    Register(ctx context.Context, req *RegisterRequest) (*User, error)
    VerifyEmail(ctx context.Context, token string) error
    ResendVerification(ctx context.Context, email string) error

    // 密码管理
    Login(ctx context.Context, email, password string) (*auth.TokenPair, *User, error)
    RequestPasswordReset(ctx context.Context, email string) error
    ResetPassword(ctx context.Context, token, newPassword string) error
    ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error

    // 用户操作
    GetUser(ctx context.Context, id uuid.UUID) (*User, error)
    UpdateProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) (*User, error)
    DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error

    // 管理员操作
    ListUsers(ctx context.Context, filter *UserFilter, pagination *Pagination) ([]*User, int64, error)
    SuspendUser(ctx context.Context, userID uuid.UUID, reason string) error
    ReactivateUser(ctx context.Context, userID uuid.UUID) error
    SetAdminStatus(ctx context.Context, userID uuid.UUID, isAdmin bool) error
}
```

### 4.2 计费服务接口

```go
// internal/module/billing/service.go

// BillingService 计费服务接口
type BillingService interface {
    // 套餐管理
    ListPlans(ctx context.Context) ([]*Plan, error)
    GetPlan(ctx context.Context, planID string) (*Plan, error)

    // 订阅管理
    GetSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error)
    CreateSubscription(ctx context.Context, userID uuid.UUID, planID string) (*Subscription, error)
    UpgradePlan(ctx context.Context, userID uuid.UUID, newPlanID string) (*Order, error)
    DowngradePlan(ctx context.Context, userID uuid.UUID, newPlanID string) (*Subscription, error)
    CancelSubscription(ctx context.Context, userID uuid.UUID, immediately bool) (*Subscription, error)

    // 配额管理
    GetQuotaStatus(ctx context.Context, userID uuid.UUID) (*QuotaStatus, error)
    CheckQuota(ctx context.Context, userID uuid.UUID, taskType string) error
    ConsumeQuota(ctx context.Context, userID uuid.UUID, tokens int) error

    // 用量统计
    GetUsageStats(ctx context.Context, userID uuid.UUID, period Period) (*UsageStats, error)
    RecordUsage(ctx context.Context, record *UsageRecord) error

    // 余额管理
    GetBalance(ctx context.Context, userID uuid.UUID) (int64, error)
    AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error
    DeductCredits(ctx context.Context, userID uuid.UUID, amount int64, reason string) error
}

// QuotaStatus 配额状态
type QuotaStatus struct {
    Plan            string    `json:"plan"`
    TokensUsed      int64     `json:"tokens_used"`
    TokensLimit     int64     `json:"tokens_limit"`
    TokensRemaining int64     `json:"tokens_remaining"`
    RequestsToday   int       `json:"requests_today"`
    RequestsLimit   int       `json:"requests_limit"`
    ResetAt         time.Time `json:"reset_at"`
}
```

### 4.3 订单服务接口

```go
// internal/module/order/service.go

// OrderService 订单服务接口
type OrderService interface {
    // 订单创建
    CreateSubscriptionOrder(ctx context.Context, userID uuid.UUID, planID string) (*Order, error)
    CreateTopupOrder(ctx context.Context, userID uuid.UUID, amount int64) (*Order, error)
    CreateUpgradeOrder(ctx context.Context, userID uuid.UUID, newPlanID string) (*Order, error)

    // 订单查询
    GetOrder(ctx context.Context, orderID uuid.UUID) (*Order, error)
    GetOrderByNo(ctx context.Context, orderNo string) (*Order, error)
    ListOrders(ctx context.Context, userID uuid.UUID, filter *OrderFilter) ([]*Order, error)

    // 订单状态
    MarkAsPaid(ctx context.Context, orderID uuid.UUID, paymentID uuid.UUID) error
    CancelOrder(ctx context.Context, orderID uuid.UUID, reason string) error
    RefundOrder(ctx context.Context, orderID uuid.UUID, amount int64, reason string) error

    // 发票
    GenerateInvoice(ctx context.Context, orderID uuid.UUID) (*Invoice, error)
    GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
    ListInvoices(ctx context.Context, userID uuid.UUID) ([]*Invoice, error)
}
```

### 4.4 支付服务接口

```go
// internal/module/payment/service.go

// PaymentService 支付服务接口
type PaymentService interface {
    // 支付创建
    CreatePaymentIntent(ctx context.Context, orderID uuid.UUID) (*PaymentIntent, error)
    ConfirmPayment(ctx context.Context, paymentIntentID string) (*Payment, error)

    // 订阅支付
    CreateSubscription(ctx context.Context, userID uuid.UUID, planID string, paymentMethodID string) (*StripeSubscription, error)
    CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error

    // 退款
    CreateRefund(ctx context.Context, paymentID uuid.UUID, amount int64, reason string) error

    // Webhook
    HandleWebhook(ctx context.Context, payload []byte, signature string) error

    // 支付方式
    AttachPaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error
    ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*PaymentMethodInfo, error)
    DetachPaymentMethod(ctx context.Context, paymentMethodID string) error
}

// PaymentProvider 支付提供商接口（策略模式）
type PaymentProvider interface {
    Name() string

    // 客户管理
    CreateCustomer(ctx context.Context, user *User) (string, error)

    // 支付
    CreatePaymentIntent(ctx context.Context, amount int64, currency string, metadata map[string]string) (*ProviderPaymentIntent, error)
    ConfirmPaymentIntent(ctx context.Context, intentID string) (*ProviderPayment, error)

    // 订阅
    CreateSubscription(ctx context.Context, customerID, priceID string) (*ProviderSubscription, error)
    CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error

    // 退款
    CreateRefund(ctx context.Context, chargeID string, amount int64) (*ProviderRefund, error)

    // Webhook
    VerifyWebhookSignature(payload []byte, signature string) error
    ParseWebhookEvent(payload []byte) (*WebhookEvent, error)
}
```

---

## 五、数据库迁移

### 5.1 用户表扩展

```sql
-- migrations/000004_extend_users.up.sql

-- 添加用户状态和认证方式字段
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS password_hash TEXT,
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS suspend_reason TEXT,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- 修改 OAuth 字段为可空
ALTER TABLE users
    ALTER COLUMN oauth_provider DROP NOT NULL,
    ALTER COLUMN oauth_id DROP NOT NULL;

-- 添加软删除索引
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- 邮箱验证表
CREATE TABLE IF NOT EXISTS email_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) NOT NULL UNIQUE,
    purpose VARCHAR(50) NOT NULL,  -- registration, password_reset
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_email_verifications_token ON email_verifications(token);
CREATE INDEX idx_email_verifications_user ON email_verifications(user_id);
```

### 5.2 计费表

```sql
-- migrations/000005_create_billing_tables.up.sql

-- 套餐表
CREATE TABLE IF NOT EXISTS plans (
    id VARCHAR(50) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    billing_cycle VARCHAR(50),
    price_usd BIGINT NOT NULL DEFAULT 0,
    stripe_price_id VARCHAR(255),
    monthly_tokens BIGINT NOT NULL DEFAULT 0,
    daily_requests INT NOT NULL DEFAULT 0,
    max_api_keys INT NOT NULL DEFAULT 3,
    features TEXT[] DEFAULT '{}',
    active BOOLEAN DEFAULT true,
    display_order INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_plans_type ON plans(type);
CREATE INDEX idx_plans_active ON plans(active) WHERE active = true;

-- 订阅表
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    plan_id VARCHAR(50) NOT NULL REFERENCES plans(id),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    cancel_at_period_end BOOLEAN DEFAULT false,
    canceled_at TIMESTAMPTZ,
    credits_balance BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_stripe ON subscriptions(stripe_subscription_id);

-- 用量记录表（TimescaleDB 超表）
CREATE TABLE IF NOT EXISTS usage_records (
    id BIGSERIAL,
    user_id UUID NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    request_id VARCHAR(255) NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    provider_id UUID NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    cost_usd DECIMAL(10, 6) NOT NULL,
    latency_ms INT NOT NULL,
    success BOOLEAN NOT NULL,
    PRIMARY KEY (id, timestamp)
);

-- 转换为 TimescaleDB 超表
SELECT create_hypertable('usage_records', 'timestamp', if_not_exists => TRUE);

-- 创建索引
CREATE INDEX idx_usage_user_time ON usage_records(user_id, timestamp DESC);
CREATE INDEX idx_usage_task_type ON usage_records(task_type, timestamp DESC);

-- 压缩策略
ALTER TABLE usage_records SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'user_id'
);
SELECT add_compression_policy('usage_records', INTERVAL '7 days', if_not_exists => TRUE);

-- 保留策略（1年）
SELECT add_retention_policy('usage_records', INTERVAL '1 year', if_not_exists => TRUE);

-- 触发器
CREATE TRIGGER update_plans_updated_at
    BEFORE UPDATE ON plans
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 5.3 订单表

```sql
-- migrations/000006_create_order_tables.up.sql

-- 订单表
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_no VARCHAR(50) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    subtotal BIGINT NOT NULL DEFAULT 0,
    discount BIGINT NOT NULL DEFAULT 0,
    tax BIGINT NOT NULL DEFAULT 0,
    total BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(10) DEFAULT 'usd',
    plan_id VARCHAR(50) REFERENCES plans(id),
    credits_amount BIGINT DEFAULT 0,
    stripe_payment_intent_id VARCHAR(255),
    stripe_invoice_id VARCHAR(255),
    paid_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    refunded_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_user ON orders(user_id, created_at DESC);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_stripe_pi ON orders(stripe_payment_intent_id);

-- 订单项表
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity INT DEFAULT 1,
    unit_price BIGINT NOT NULL,
    amount BIGINT NOT NULL
);

CREATE INDEX idx_order_items_order ON order_items(order_id);

-- 发票表
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_no VARCHAR(50) NOT NULL UNIQUE,
    order_id UUID NOT NULL REFERENCES orders(id),
    user_id UUID NOT NULL REFERENCES users(id),
    amount BIGINT NOT NULL,
    currency VARCHAR(10) DEFAULT 'usd',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    pdf_url TEXT,
    stripe_invoice_id VARCHAR(255),
    issued_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    due_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoices_user ON invoices(user_id);
CREATE INDEX idx_invoices_order ON invoices(order_id);

-- 触发器
CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 5.4 支付表

```sql
-- migrations/000007_create_payment_tables.up.sql

-- 支付记录表
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    amount BIGINT NOT NULL,
    currency VARCHAR(10) DEFAULT 'usd',
    method VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    provider VARCHAR(50) NOT NULL DEFAULT 'stripe',
    stripe_payment_intent_id VARCHAR(255),
    stripe_charge_id VARCHAR(255),
    failure_code VARCHAR(100),
    failure_message TEXT,
    refunded_amount BIGINT DEFAULT 0,
    succeeded_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payments_order ON payments(order_id);
CREATE INDEX idx_payments_user ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_stripe_pi ON payments(stripe_payment_intent_id);

-- Stripe Webhook 事件表（幂等性）
CREATE TABLE IF NOT EXISTS stripe_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(255) NOT NULL,
    data JSONB NOT NULL,
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhook_events_type ON stripe_webhook_events(type);
CREATE INDEX idx_webhook_events_processed ON stripe_webhook_events(processed) WHERE processed = false;

-- 触发器
CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 5.5 初始套餐数据

```sql
-- migrations/000008_seed_plans.up.sql

INSERT INTO plans (id, type, name, description, billing_cycle, price_usd, monthly_tokens, daily_requests, max_api_keys, features, display_order) VALUES
('free', 'free', 'Free', '免费体验计划', NULL, 0, 10000, 100, 2,
 ARRAY['10K tokens/月', '100 请求/天', '基础模型访问'], 1),

('pro_monthly', 'pro', 'Pro Monthly', '专业版月付', 'monthly', 2000, 500000, 2000, 5,
 ARRAY['500K tokens/月', '2000 请求/天', '所有模型访问', '优先支持'], 2),

('pro_yearly', 'pro', 'Pro Yearly', '专业版年付', 'yearly', 20000, 500000, 2000, 5,
 ARRAY['500K tokens/月', '2000 请求/天', '所有模型访问', '优先支持', '2个月免费'], 3),

('team_monthly', 'team', 'Team Monthly', '团队版月付', 'monthly', 5000, 2000000, 10000, 20,
 ARRAY['2M tokens/月', '10000 请求/天', '团队协作', 'API 优先级', '专属支持'], 4),

('team_yearly', 'team', 'Team Yearly', '团队版年付', 'yearly', 50000, 2000000, 10000, 20,
 ARRAY['2M tokens/月', '10000 请求/天', '团队协作', 'API 优先级', '专属支持', '2个月免费'], 5),

('enterprise', 'enterprise', 'Enterprise', '企业版', NULL, -1, -1, -1, -1,
 ARRAY['无限 tokens', '无限请求', '定制化部署', 'SLA 保障', '专属客户经理'], 6);
```

---

## 六、实施阶段

### P0 阶段：核心计费（5天）

| 任务 | 文件 | 工作量 |
|------|------|--------|
| User 模型扩展 | `user/model.go` | 0.5d |
| 用户状态管理 | `user/service.go` | 1d |
| 套餐和订阅模型 | `billing/model.go` | 0.5d |
| 订阅 Repository | `billing/repository.go` | 0.5d |
| 配额管理器 | `billing/quota/manager.go` | 1d |
| 用量记录 | `billing/usage/recorder.go` | 0.5d |
| 数据库迁移 | `migrations/000004-000008` | 1d |

**产出**: 基本的订阅管理和配额检查能力

### P1 阶段：支付集成（5天）

| 任务 | 文件 | 工作量 |
|------|------|--------|
| 邮箱注册+验证 | `user/verification.go` | 1d |
| 订单模型和服务 | `order/service.go` | 1d |
| Stripe Provider | `payment/provider/stripe.go` | 1.5d |
| Webhook 处理 | `payment/webhook_handler.go` | 1d |
| API 接口 | `*/handler.go` | 0.5d |

**产出**: 完整的订阅支付流程

### P2 阶段：完善功能（3天）

| 任务 | 文件 | 工作量 |
|------|------|--------|
| 管理员用户管理 | `user/admin_handler.go` | 1d |
| 发票生成 | `order/invoice.go` | 0.5d |
| 退款流程 | `payment/service.go` | 0.5d |
| 用量统计报表 | `billing/handler.go` | 1d |

**产出**: 管理功能和报表

---

## 七、API 接口设计

### 7.1 用户 API

```yaml
# 邮箱注册
POST /api/v1/auth/register:
  request:
    email: string (required)
    password: string (required, min 8 chars)
    name: string (required)
  response:
    user: UserResponse
    message: "Verification email sent"

# 验证邮箱
POST /api/v1/auth/verify-email:
  request:
    token: string (required)
  response:
    message: "Email verified successfully"

# 密码登录
POST /api/v1/auth/login/password:
  request:
    email: string (required)
    password: string (required)
  response:
    tokens: TokenPair
    user: UserResponse

# 用户注销
DELETE /api/v1/users/me:
  request:
    password: string (required)
  response:
    message: "Account deleted successfully"

# 管理员 - 暂停用户
POST /api/v1/admin/users/{id}/suspend:
  request:
    reason: string (required)
  response:
    user: UserResponse

# 管理员 - 恢复用户
POST /api/v1/admin/users/{id}/reactivate:
  response:
    user: UserResponse
```

### 7.2 计费 API

```yaml
# 获取套餐列表
GET /api/v1/billing/plans:
  response:
    plans: Plan[]

# 获取当前订阅
GET /api/v1/billing/subscription:
  response:
    subscription: Subscription
    plan: Plan
    quota: QuotaStatus

# 获取配额状态
GET /api/v1/billing/quota:
  response:
    plan: string
    tokens_used: int
    tokens_limit: int
    tokens_remaining: int
    requests_today: int
    requests_limit: int
    reset_at: timestamp

# 获取用量统计
GET /api/v1/billing/usage:
  query:
    period: day | week | month
    start_date: date (optional)
    end_date: date (optional)
  response:
    total_tokens: int
    total_requests: int
    total_cost_usd: float
    by_model: {...}
    by_day: [...]
```

### 7.3 订单 API

```yaml
# 创建订阅订单
POST /api/v1/orders/subscription:
  request:
    plan_id: string (required)
  response:
    order: Order
    payment_intent: PaymentIntent

# 创建充值订单
POST /api/v1/orders/topup:
  request:
    amount: int (required, in cents)
  response:
    order: Order
    payment_intent: PaymentIntent

# 获取订单列表
GET /api/v1/orders:
  query:
    status: string (optional)
    type: string (optional)
    page: int
    page_size: int
  response:
    orders: Order[]
    total: int

# 获取订单详情
GET /api/v1/orders/{id}:
  response:
    order: Order
    items: OrderItem[]
    payments: Payment[]
```

### 7.4 支付 API

```yaml
# 确认支付
POST /api/v1/payments/confirm:
  request:
    payment_intent_id: string (required)
  response:
    payment: Payment

# 获取支付方式列表
GET /api/v1/payments/methods:
  response:
    methods: PaymentMethodInfo[]

# 添加支付方式
POST /api/v1/payments/methods:
  request:
    payment_method_id: string (required)
  response:
    method: PaymentMethodInfo

# Stripe Webhook
POST /api/v1/webhooks/stripe:
  headers:
    Stripe-Signature: string
  response:
    received: true
```

---

## 八、关键实现

### 8.1 配额检查中间件

```go
// internal/module/billing/quota/checker.go

type QuotaChecker struct {
    billing BillingService
}

func NewQuotaChecker(billing BillingService) *QuotaChecker {
    return &QuotaChecker{billing: billing}
}

func (c *QuotaChecker) Middleware() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        userID := ctx.MustGet("user_id").(uuid.UUID)

        // 从请求中确定任务类型
        taskType := determineTaskType(ctx)

        if err := c.billing.CheckQuota(ctx.Request.Context(), userID, taskType); err != nil {
            if errors.Is(err, ErrQuotaExceeded) {
                ctx.JSON(http.StatusPaymentRequired, gin.H{
                    "error": "quota_exceeded",
                    "message": "Your quota has been exceeded. Please upgrade your plan.",
                })
                ctx.Abort()
                return
            }
            ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
            ctx.Abort()
            return
        }

        ctx.Next()
    }
}
```

### 8.2 订单状态机

```go
// internal/module/order/state_machine.go

type OrderStateMachine struct {
    transitions map[OrderStatus][]OrderStatus
}

func NewOrderStateMachine() *OrderStateMachine {
    return &OrderStateMachine{
        transitions: map[OrderStatus][]OrderStatus{
            OrderStatusPending:  {OrderStatusPaid, OrderStatusCanceled, OrderStatusFailed},
            OrderStatusPaid:     {OrderStatusRefunded},
            OrderStatusCanceled: {},
            OrderStatusRefunded: {},
            OrderStatusFailed:   {OrderStatusPending}, // 可重试
        },
    }
}

func (sm *OrderStateMachine) CanTransition(from, to OrderStatus) bool {
    allowed, ok := sm.transitions[from]
    if !ok {
        return false
    }
    for _, s := range allowed {
        if s == to {
            return true
        }
    }
    return false
}

func (sm *OrderStateMachine) Transition(order *Order, to OrderStatus) error {
    if !sm.CanTransition(order.Status, to) {
        return fmt.Errorf("invalid transition from %s to %s", order.Status, to)
    }
    order.Status = to
    return nil
}
```

### 8.3 Stripe Webhook 处理

```go
// internal/module/payment/webhook_handler.go

type WebhookHandler struct {
    payment   PaymentService
    order     OrderService
    billing   BillingService
    eventRepo WebhookEventRepository
}

func (h *WebhookHandler) Handle(ctx context.Context, event *stripe.Event) error {
    // 幂等性检查
    if exists, _ := h.eventRepo.Exists(ctx, event.ID); exists {
        return nil // 已处理过
    }

    // 记录事件
    if err := h.eventRepo.Create(ctx, &StripeWebhookEvent{
        EventID: event.ID,
        Type:    event.Type,
        Data:    string(event.Data.Raw),
    }); err != nil {
        return fmt.Errorf("record event: %w", err)
    }

    var err error
    switch event.Type {
    case "payment_intent.succeeded":
        err = h.handlePaymentSucceeded(ctx, event)
    case "payment_intent.payment_failed":
        err = h.handlePaymentFailed(ctx, event)
    case "customer.subscription.created":
        err = h.handleSubscriptionCreated(ctx, event)
    case "customer.subscription.updated":
        err = h.handleSubscriptionUpdated(ctx, event)
    case "customer.subscription.deleted":
        err = h.handleSubscriptionDeleted(ctx, event)
    case "invoice.paid":
        err = h.handleInvoicePaid(ctx, event)
    case "invoice.payment_failed":
        err = h.handleInvoicePaymentFailed(ctx, event)
    }

    // 标记为已处理
    h.eventRepo.MarkProcessed(ctx, event.ID, err)

    return err
}

func (h *WebhookHandler) handlePaymentSucceeded(ctx context.Context, event *stripe.Event) error {
    var pi stripe.PaymentIntent
    if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
        return err
    }

    // 获取订单
    orderID := pi.Metadata["order_id"]
    order, err := h.order.GetOrder(ctx, uuid.MustParse(orderID))
    if err != nil {
        return err
    }

    // 更新订单状态
    if err := h.order.MarkAsPaid(ctx, order.ID, uuid.Nil); err != nil {
        return err
    }

    // 如果是充值订单，增加余额
    if order.Type == OrderTypeTopup {
        return h.billing.AddCredits(ctx, order.UserID, order.CreditsAmount, "topup")
    }

    return nil
}
```

---

## 九、配置扩展

```yaml
# config.yaml 新增配置

billing:
  stripe:
    api_key: ${STRIPE_API_KEY}
    webhook_secret: ${STRIPE_WEBHOOK_SECRET}
    publishable_key: ${STRIPE_PUBLISHABLE_KEY}

email:
  smtp_host: ${SMTP_HOST}
  smtp_port: ${SMTP_PORT}
  smtp_user: ${SMTP_USER}
  smtp_password: ${SMTP_PASSWORD}
  from_address: ${EMAIL_FROM}
  from_name: "UniEdit"
```

---

## 十、验收标准

- [ ] 邮箱注册和验证流程完整
- [ ] 用户状态管理（active/suspended/deleted）
- [ ] 订阅升级/降级/取消
- [ ] 配额检查阻止超额请求
- [ ] Stripe 订阅支付成功
- [ ] 余额充值功能
- [ ] 管理员可启用/禁用用户
- [ ] 单元测试覆盖率 > 70%

---

## 十一、关键文件参考

| 文件 | 用途 |
|------|------|
| `internal/module/auth/model.go` | 参考 User 模型结构 |
| `internal/module/auth/service.go` | 参考认证服务实现 |
| `internal/module/ai/module.go` | 参考模块初始化模式 |
| `internal/shared/errors/errors.go` | 参考错误处理模式 |
| `docs/design-p0-core.md:1702-1936` | Billing 模块设计文档 |
