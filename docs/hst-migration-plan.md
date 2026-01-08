# HST DDD 架构迁移计划

## 当前状态

项目已部分采用 DDD 模式：
- ✅ Order, Billing, Payment 模块已有 `domain/` 和 `entity/` 分层
- ✅ 领域实体使用充血模型（含业务方法）
- ✅ Repository 接口与实现分离

## 目标架构

```
internal/
├── domain/                  # 【领域层】集中管理所有业务核心
│   ├── billing/            # 限界上下文
│   │   ├── plan.go         # 领域实体
│   │   ├── subscription.go
│   │   ├── repository.go   # 端口（接口定义）
│   │   └── errors.go       # 领域错误
│   ├── order/
│   ├── payment/
│   ├── user/
│   └── auth/
├── app/                     # 【应用层】CQRS 命令/查询处理
│   ├── command/            # 写操作
│   │   ├── billing/
│   │   │   ├── create_subscription.go
│   │   │   └── cancel_subscription.go
│   │   └── order/
│   └── query/              # 读操作（可绕过领域层优化性能）
│       ├── billing/
│       │   └── get_usage_stats.go
│       └── order/
├── infra/                   # 【基础设施层】技术实现
│   ├── persistence/        # Repository 实现（适配器）
│   │   ├── billing_repo.go
│   │   ├── order_repo.go
│   │   └── payment_repo.go
│   ├── external/           # 外部服务客户端
│   │   ├── stripe/
│   │   ├── alipay/
│   │   └── wechat/
│   └── cache/
├── ports/                   # 【接口层】外部适配器
│   ├── http/               # REST API
│   │   ├── billing_handler.go
│   │   ├── order_handler.go
│   │   └── middleware/
│   └── grpc/               # gRPC（未来）
├── module/                  # 【过渡】保留现有模块（逐步迁移）
│   └── ai/                 # AI 模块暂不迁移
└── shared/                  # 共享工具
```

## 迁移步骤

### Phase 1: 建立 domain/ 层（已完成）
- [x] 创建 `internal/domain/{billing,order,payment}/`
- [x] 迁移领域实体（无 GORM tag）
- [x] 定义 Repository 接口（端口）
- [x] 定义领域错误
- [x] 创建 `internal/domain/user/` (user.go, verification.go, repository.go, errors.go)

### Phase 2: 建立 infra/ 层（已完成）
- [x] 创建 `infra/persistence/`
- [x] 迁移 Repository 实现（billing_repo, order_repo, payment_repo）
- [x] 创建 Entity 层（infra/persistence/entity/）
- [x] 创建 user_repo.go 和 user_entity.go
- [ ] 创建 `infra/external/` 放置第三方客户端（待需要时添加）

### Phase 3: 建立 ports/ 层（已完成）
- [x] 创建 `ports/http/`
- [x] 创建通用工具（common.go: getUserID, 分页, 响应辅助）
- [x] 创建统一错误处理（error_handler.go）
- [x] 创建 billing_handler.go（CQRS 模式）
- [x] 创建 order_handler.go（CQRS 模式）
- [x] 创建 payment_handler.go（CQRS 模式）
- [x] 创建 user_handler.go（CQRS 模式）

### Phase 4: 建立 app/ 层（CQRS）（已完成）
- [x] 创建 Command 处理器（billing, order, payment）
- [x] 创建 Query 处理器（billing, order, payment）
- [x] 定义通用 Handler 接口
- [x] 创建 Wire 依赖注入配置（infra/wire/wire.go）
- [x] 添加 HTTP Handler Wire 集成（HSTHandlers, InitializeHSTHandlers）
- [x] 创建 User Command 处理器（registration.go, profile.go）
- [x] 创建 User Query 处理器（get_user.go）
- [x] 添加 User 到 Wire 配置（UserCommandSet, UserQuerySet, UserHandler）

### Phase 5: 清理
- [ ] 删除 `module/` 中已迁移的模块
- [ ] 更新所有 import 路径

## 关键原则

1. **依赖方向**：外层 → 内层（ports → app → domain ← infra）
2. **领域干净**：domain 无技术依赖（无 GORM/JSON tag）
3. **接口隔离**：接口定义在使用方（domain 定义，infra 实现）
4. **CQRS 分离**：写操作走 Command，读操作走 Query（可优化）

## 当前已创建的结构

```
internal/
├── domain/
│   ├── billing/  (plan, subscription, usage, repository, errors, status)
│   ├── order/    (order, money, status, repository, errors, invoice)
│   ├── payment/  (payment, status, webhook_event, repository, errors)
│   ├── user/     (user, verification, repository, errors)
│   └── auth/     (oauth, refresh_token, user_api_key, system_api_key, token_pair, repository, errors, api_key_util) ← 新增
├── infra/
│   ├── persistence/
│   │   ├── entity/
│   │   │   ├── billing_entity.go
│   │   │   ├── order_entity.go
│   │   │   ├── payment_entity.go
│   │   │   ├── user_entity.go
│   │   │   └── auth_entity.go      ← 新增 (RefreshToken, UserAPIKey, SystemAPIKey)
│   │   ├── billing_repo.go
│   │   ├── order_repo.go
│   │   ├── payment_repo.go
│   │   ├── user_repo.go
│   │   └── auth_repo.go            ← 新增 (RefreshToken, UserAPIKey, SystemAPIKey Repos)
│   ├── wire/
│   │   └── wire.go               (Wire DI 配置，含 HSTHandlers, AuthHandlers)
│   ├── external/     (待填充)
│   └── clients/      (待填充)
├── ports/
│   ├── http/
│   │   ├── common.go             (共享工具：getUserID, 分页, 响应)
│   │   ├── error_handler.go      (统一错误处理)
│   │   ├── billing_handler.go    (Billing CQRS Handler)
│   │   ├── order_handler.go      (Order CQRS Handler)
│   │   ├── payment_handler.go    (Payment CQRS Handler)
│   │   ├── user_handler.go       (User CQRS Handler)
│   │   └── auth_handler.go       (Auth CQRS Handler + Middlewares) ← 新增
│   └── grpc/   (待填充)
└── app/
    ├── command/
    │   ├── handler.go            (通用 Handler 接口)
    │   ├── billing/
    │   │   ├── create_subscription.go
    │   │   └── cancel_subscription.go
    │   ├── order/
    │   │   ├── create_order.go
    │   │   └── cancel_order.go
    │   ├── payment/
    │   │   ├── create_payment.go
    │   │   ├── mark_succeeded.go
    │   │   └── refund_payment.go
    │   ├── user/
    │   │   ├── registration.go   (Register, VerifyEmail, ResendVerification, PasswordReset)
    │   │   └── profile.go        (UpdateProfile, DeleteAccount, Suspend, Reactivate, SetAdmin)
    │   └── auth/                 ← 新增
    │       ├── oauth.go          (InitiateOAuth)
    │       ├── token.go          (CompleteOAuth, RefreshTokens, Logout)
    │       ├── user_api_key.go   (Create, Delete, Rotate, GetDecrypted)
    │       └── system_api_key.go (Create, Update, Delete, Rotate, Validate)
    └── query/
        ├── handler.go            (通用 Handler 接口)
        ├── billing/
        │   ├── get_subscription.go
        │   ├── list_plans.go
        │   └── get_usage_stats.go
        ├── order/
        │   ├── get_order.go
        │   └── list_orders.go (in get_order.go)
        ├── payment/
        │   └── get_payment.go    (含 ListPaymentsByOrder)
        ├── user/
        │   └── get_user.go       (GetUser, GetUserByEmail, ListUsers)
        └── auth/                 ← 新增
            └── api_keys.go       (ListUserAPIKeys, ListSystemAPIKeys, GetSystemAPIKey)
```

## Wire 依赖注入使用示例

```go
// 在 main.go 或 app.go 中初始化 HST handlers
handlers, err := wire.InitializeHSTHandlers(db)
if err != nil {
    log.Fatal(err)
}

// 注册路由
api := r.Group("/api/v1")

// Public routes
handlers.Billing.RegisterRoutes(api)
handlers.User.RegisterRoutes(api)

// Protected routes (需要认证)
handlers.Billing.RegisterProtectedRoutes(api)
handlers.Order.RegisterProtectedRoutes(api)
handlers.Payment.RegisterProtectedRoutes(api)
handlers.User.RegisterProtectedRoutes(api)

// Admin routes
handlers.Payment.RegisterAdminRoutes(adminAPI)
handlers.User.RegisterAdminRoutes(adminAPI)
```

## 已迁移模块

| 模块 | 领域层 | 基础设施层 | 应用层 | 接口层 | 状态 |
|------|--------|-----------|--------|--------|------|
| Billing | ✅ | ✅ | ✅ | ✅ | 已完成 |
| Order | ✅ | ✅ | ✅ | ✅ | 已完成 |
| Payment | ✅ | ✅ | ✅ | ✅ | 已完成 |
| User | ✅ | ✅ | ✅ | ✅ | 已完成 |
| Auth | ✅ | ✅ | ✅ | ✅ | 已完成 |
| Collaboration | ⏳ | ⏳ | ⏳ | ⏳ | 待迁移 |
| Git | ⏳ | ⏳ | ⏳ | ⏳ | 待迁移 |
| AI | ⏳ | ⏳ | ⏳ | ⏳ | 待迁移 |

## 迁移策略

### 渐进式迁移（推荐）

由于现有 `module/` 中的代码已经在生产使用，完整迁移需要：

1. **保持双轨运行**：新的 HST 结构与现有 module 并存
2. **新功能使用 HST**：所有新增功能采用 HST 架构
3. **逐步替换**：在重构或修改时，将 module 代码迁移到 HST 层
4. **最终清理**：当所有代码迁移完成后，删除 module 目录

### 依赖关系

```
ports/http/handler.go
    ↓ 依赖
app/command/ 或 app/query/
    ↓ 依赖
domain/ (接口定义)
    ↑ 实现
infra/persistence/ (Repository 实现)
```
