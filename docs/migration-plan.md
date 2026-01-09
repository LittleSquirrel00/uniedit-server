# 架构迁移方案：从 module 到 domain+adapter

## 1. 当前状态分析

### 1.1 现有架构
```
internal/
├── module/           # 旧架构（~32k 行代码）
│   ├── ai/           # AI 模块
│   ├── auth/         # 认证模块
│   ├── billing/      # 计费模块
│   ├── collaboration/# 协作模块
│   ├── git/          # Git 托管模块
│   ├── order/        # 订单模块
│   ├── payment/      # 支付模块
│   └── user/         # 用户模块
├── domain/           # 新架构 - 领域层（已完成）
│   ├── ai/           # ✅ 测试覆盖 77.9%
│   ├── auth/         # ✅ 测试覆盖 80.5%
│   ├── billing/      # ✅ 测试覆盖 81.0%
│   ├── collaboration/# ✅ 测试覆盖 74.5%
│   ├── git/          # ✅ 测试覆盖 54.9%
│   ├── media/        # ✅ 测试覆盖 79.2%
│   ├── order/        # ✅ 测试覆盖 88.5%
│   ├── payment/      # ✅ 测试覆盖 83.4%
│   └── user/         # ✅ 测试覆盖 80.4%
├── port/             # 新架构 - 端口定义（已完成）
│   ├── inbound/      # 入站端口接口
│   └── outbound/     # 出站端口接口
├── adapter/          # 新架构 - 适配器层（部分完成）
│   ├── inbound/      # HTTP handlers
│   ├── outbound/     # 数据库、缓存、第三方服务
│   ├── database/     # Git 数据库适配器
│   ├── storage/      # Git 存储适配器
│   └── http/         # Git/Media/Collaboration HTTP 处理器
└── app/              # 应用组装层（仍使用旧 module）
    └── app.go        # ❌ 直接依赖 internal/module/*
```

### 1.2 迁移目标
```
internal/
├── domain/           # 业务逻辑（纯净，无框架依赖）
├── port/             # 端口接口定义
├── adapter/          # 适配器实现
│   ├── inbound/      # HTTP/gRPC 入站
│   └── outbound/     # 数据库/缓存/第三方 出站
├── app/              # 应用组装
│   └── app.go        # 使用新架构组装
└── shared/           # 共享基础设施（保留）
```

### 1.3 依赖关系

**当前 app.go 依赖的旧模块：**
1. `module/ai` - AI 模块（复杂，包含 provider/pool/cache 子模块）
2. `module/auth` - 认证模块
3. `module/billing` - 计费模块（包含 quota/usage 子模块）
4. `module/collaboration` - 协作模块
5. `module/git` - Git 模块（包含 lfs/storage 子模块）
6. `module/order` - 订单模块
7. `module/payment` - 支付模块（包含 provider 子模块）
8. `module/user` - 用户模块

**已完成的新架构适配器：**
- ✅ `adapter/inbound/http/ai/` - AI HTTP 处理器
- ✅ `adapter/http/git_handler.go` - Git HTTP 处理器
- ✅ `adapter/http/media_handler.go` - Media HTTP 处理器
- ✅ `adapter/http/collaboration_handler.go` - 协作 HTTP 处理器
- ✅ `adapter/inbound/gin/` - 用户/认证/计费等 Gin 处理器
- ✅ `adapter/outbound/postgres/` - 数据库适配器
- ✅ `adapter/outbound/redis/` - 缓存适配器
- ✅ `adapter/outbound/aiprovider/` - AI 供应商适配器

## 2. 迁移策略

### 2.1 分阶段迁移

采用"绞杀者模式"（Strangler Fig Pattern）进行渐进式迁移：

```
Phase 1: 基础设施准备
├── 创建新的 app_v2.go
├── 实现共用的工厂函数
└── 建立并行运行机制

Phase 2: 逐模块迁移（按依赖顺序）
├── User 模块（无依赖）
├── Auth 模块（依赖 User）
├── Billing 模块（依赖 Auth）
├── Order 模块（依赖 Billing）
├── Payment 模块（依赖 Order）
├── AI 模块（依赖 Billing）
├── Git 模块（独立）
├── Collaboration 模块（依赖 Git）
└── Media 模块（依赖 AI 基础设施）

Phase 3: 清理旧代码
├── 删除 internal/module/
├── 更新导入路径
└── 最终验证
```

### 2.2 模块迁移顺序（依赖拓扑排序）

```
Level 0（无依赖）:
  - User
  - Git

Level 1:
  - Auth (依赖 User)
  - Collaboration (依赖 Git)

Level 2:
  - Billing (依赖 Auth)

Level 3:
  - Order (依赖 Billing)
  - AI (依赖 Billing)

Level 4:
  - Payment (依赖 Order, Billing)
  - Media (依赖 AI 基础设施)
```

## 3. 实施细节

### 3.1 创建新应用组装器

```go
// internal/app/app_v2.go
package app

import (
    // 新架构导入
    "github.com/uniedit/server/internal/domain/ai"
    "github.com/uniedit/server/internal/domain/auth"
    "github.com/uniedit/server/internal/domain/billing"
    // ... 其他 domain

    aihttp "github.com/uniedit/server/internal/adapter/inbound/http/ai"
    githttp "github.com/uniedit/server/internal/adapter/http"
    // ... 其他适配器

    "github.com/uniedit/server/internal/adapter/outbound/postgres"
    "github.com/uniedit/server/internal/adapter/outbound/redis"
    "github.com/uniedit/server/internal/adapter/outbound/aiprovider"
)

type AppV2 struct {
    config    *config.Config
    db        *gorm.DB
    redis     redis.UniversalClient
    router    *gin.Engine
    logger    *zap.Logger

    // Domains (业务逻辑)
    aiDomain      ai.AIDomain
    authDomain    auth.AuthDomain
    billingDomain billing.BillingDomain
    // ...

    // HTTP Handlers (入站适配器)
    aiChatHandler    *aihttp.ChatHandler
    gitHandler       *githttp.GitHandler
    // ...
}

func NewV2(cfg *config.Config) (*AppV2, error) {
    // 1. 初始化基础设施
    // 2. 创建出站适配器（数据库、缓存）
    // 3. 创建领域服务
    // 4. 创建入站适配器（HTTP handlers）
    // 5. 注册路由
}
```

### 3.2 各模块迁移检查清单

#### User 模块
- [ ] 确认 `domain/user/domain.go` 实现完整
- [ ] 确认 `adapter/inbound/gin/user*.go` 使用 domain 接口
- [ ] 确认 `adapter/outbound/postgres/user*.go` 实现 outbound 端口
- [ ] 更新 app_v2.go 中的组装代码
- [ ] 验证测试通过

#### Auth 模块
- [ ] 确认 `domain/auth/domain.go` 实现完整
- [ ] 确认 `adapter/inbound/gin/auth.go` 使用 domain 接口
- [ ] 确认 `adapter/outbound/postgres/` 相关适配器
- [ ] 确认 `adapter/outbound/redis/` 相关适配器
- [ ] 确认 `adapter/outbound/oauth/` 适配器
- [ ] 更新 app_v2.go 中的组装代码
- [ ] 验证测试通过

#### Billing 模块
- [ ] 确认 `domain/billing/domain.go` 实现完整
- [ ] 确认入站适配器
- [ ] 确认出站适配器
- [ ] 更新 app_v2.go
- [ ] 验证测试通过

#### Order 模块
- [ ] 确认 `domain/order/domain.go` 实现完整
- [ ] 确认入站适配器
- [ ] 确认出站适配器
- [ ] 更新 app_v2.go
- [ ] 验证测试通过

#### Payment 模块
- [ ] 确认 `domain/payment/domain.go` 实现完整
- [ ] 确认入站适配器
- [ ] 确认出站适配器
- [ ] 更新 app_v2.go
- [ ] 验证测试通过

#### AI 模块
- [ ] 确认 `domain/ai/domain.go` 实现完整
- [ ] 确认 `adapter/inbound/http/ai/*.go` 处理器
- [ ] 确认 `adapter/outbound/aiprovider/*.go` 适配器
- [ ] 确认 `adapter/outbound/postgres/ai*.go` 适配器
- [ ] 确认 `adapter/outbound/redis/ai*.go` 适配器
- [ ] 更新 app_v2.go
- [ ] 验证测试通过

#### Git 模块
- [ ] 确认 `domain/git/*.go` 实现完整
- [ ] 确认 `adapter/http/git_handler.go` 使用 domain 接口
- [ ] 确认 `adapter/database/git*.go` 适配器
- [ ] 确认 `adapter/storage/git*.go` 适配器
- [ ] 更新 app_v2.go
- [ ] 验证测试通过

#### Collaboration 模块
- [ ] 确认 `domain/collaboration/domain.go` 实现完整
- [ ] 确认 `adapter/http/collaboration_handler.go`
- [ ] 确认 `adapter/outbound/postgres/collaboration.go`
- [ ] 更新 app_v2.go
- [ ] 验证测试通过

#### Media 模块
- [ ] 确认 `domain/media/domain.go` 实现完整
- [ ] 确认 `adapter/http/media_handler.go`
- [ ] 确认 `adapter/outbound/postgres/media.go`
- [ ] 确认 `adapter/outbound/mediaprovider/*.go`
- [ ] 确认 `adapter/outbound/redis/media_health.go`
- [ ] 更新 app_v2.go
- [ ] 验证测试通过

## 4. 风险与缓解

### 4.1 风险识别

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 接口不兼容 | 高 | 在迁移前全面对比新旧接口 |
| 功能遗漏 | 高 | 使用 deadcode 工具确保无遗漏 |
| 性能退化 | 中 | 添加基准测试对比 |
| 配置不一致 | 中 | 统一配置管理 |

### 4.2 回滚策略

1. 保持旧代码在 `internal/module/` 目录
2. 使用特性开关控制新旧架构切换
3. 每个模块单独可回滚

```go
// 特性开关示例
if cfg.Features.UseNewArchitecture {
    return NewV2(cfg)
}
return New(cfg) // 旧版本
```

## 5. 时间估算

| 阶段 | 工作项 | 复杂度 |
|------|--------|--------|
| Phase 1 | 基础设施准备 | 低 |
| Phase 2.1 | User + Git 模块 | 中 |
| Phase 2.2 | Auth + Collaboration | 中 |
| Phase 2.3 | Billing | 中 |
| Phase 2.4 | Order + AI | 高 |
| Phase 2.5 | Payment + Media | 高 |
| Phase 3 | 清理旧代码 | 低 |

## 6. 验证标准

### 6.1 功能验证
- [ ] 所有 API 端点正常工作
- [ ] 所有单元测试通过
- [ ] 集成测试通过

### 6.2 质量验证
- [ ] `go build ./...` 成功
- [ ] `go test ./...` 100% 通过
- [ ] `golangci-lint run` 无新警告
- [ ] 测试覆盖率保持或提升

### 6.3 性能验证
- [ ] 关键 API 延迟无退化
- [ ] 内存使用无异常增长

## 7. 下一步行动

1. **立即开始**：创建 `app_v2.go` 基础框架
2. **第一批迁移**：User 和 Git 模块（最独立）
3. **验证方法**：并行运行新旧版本进行对比测试
