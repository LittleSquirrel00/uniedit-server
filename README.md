# UniEdit Server

UniEdit 视频编辑器的后端服务，提供用户认证、AI 代理、计费管理、工作流仓库、Git 托管等能力。

## 技术栈

| 层级 | 技术 |
|------|------|
| 语言 | Go 1.23+ |
| 框架 | Gin (HTTP) + GORM (ORM) |
| 数据库 | PostgreSQL 16+ |
| 缓存 | Redis 7+ |
| 对象存储 | Cloudflare R2 (S3 兼容) |
| 构建工具 | Mage + Wire |

## 架构设计

项目采用**实用主义 DDD（Pragmatic DDD）**混合架构：

- **复杂模块**（AI/Billing）→ DDD 建模，领域知识显式化
- **简单模块**（Auth/Workflow）→ MVC 风格，快速开发

### 分层架构

```
┌─────────────────────────────────────────────────────────┐
│                   Interface Layer                        │
│  [HTTP Handlers]  [Middleware]  [Routes]                │
├─────────────────────────────────────────────────────────┤
│                  Application Layer                       │
│  [Services]  [Use Cases]  [DTO]                         │
├─────────────────────────────────────────────────────────┤
│                    Domain Layer                          │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │
│  │ AI Context  │ │Task Context │ │Group Context│        │
│  │             │ │             │ │             │        │
│  │ - Provider  │ │ - Task      │ │ - Group     │        │
│  │ - Model     │ │ - Status    │ │ - Strategy  │        │
│  │ - Adapter   │ │ - Executor  │ │ - Fallback  │        │
│  │ - Routing   │ │ - Poller    │ │             │        │
│  └─────────────┘ └─────────────┘ └─────────────┘        │
├─────────────────────────────────────────────────────────┤
│                 Infrastructure Layer                     │
│  [Repository]  [Cache]  [HTTP Client]  [Config]         │
└─────────────────────────────────────────────────────────┘
```

### 核心设计模式

| 模式 | 应用场景 | 实现 |
|------|----------|------|
| **Registry** | Provider/Adapter 管理 | 内存缓存 + 多维索引 |
| **Strategy Chain** | 智能路由决策 | 6 种策略按优先级链式执行 |
| **Adapter** | LLM 提供商适配 | OpenAI/Anthropic/Generic |
| **Repository** | 数据访问抽象 | 接口定义在使用方 |
| **Circuit Breaker** | 故障熔断 | gobreaker 实现 |

## 项目结构

```
uniedit-server/
├── cmd/server/            # 程序入口
├── api/
│   ├── protobuf_spec/     # Proto 接口定义 (auth/user/ai/billing/payment/order/git/media/collaboration)
│   ├── pb/                # 生成的 Go 代码 (*_pb.go, *_gin.pb.go)
│   └── openapi_spec/      # 生成的 OpenAPI v2 文档
├── internal/
│   ├── app/               # Wire 依赖注入、应用组装、路由注册
│   ├── adapter/           # 适配层：
│   │   ├── inbound/http/  # HTTP Handler (xxxproto/ 使用 Proto 消息)
│   │   └── outbound/      # Postgres/Redis/OAuth/第三方 Provider 适配器
│   ├── domain/            # 领域层 (ai/auth/billing/order/payment/git/collaboration/media/user)
│   ├── infra/             # 基础设施封装 (config/database/cache/httpclient)
│   ├── port/              # 端口定义 (inbound/outbound 接口)
│   ├── model/             # 数据库模型 (GORM)
│   ├── transport/         # 传输层工具 (protohttp 绑定)
│   └── utils/             # 通用工具 (logger/metrics/middleware 等)
├── configs/               # 配置文件模板
├── migrations/            # 数据库迁移
├── docs/                  # 设计文档
└── openspec/              # OpenSpec 规范
```

## 快速开始

### 环境要求

- Go 1.23+
- PostgreSQL 16+
- Redis 7+
- Mage (构建工具)

### 安装开发工具

```bash
# 运行设置脚本（安装 wire, mage, golangci-lint 等）
./scripts/setup.sh

# 或手动安装
go install github.com/magefile/mage@latest
go install github.com/google/wire/cmd/wire@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 或使用 mage install 安装所有开发工具
mage install
```

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/uniedit/server.git
cd uniedit-server

# 复制配置文件
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml 配置数据库和 Redis
# 如需调用 /api/v1 下的管理接口（如 /admin/ai/*、/billing/credits、用户管理等），请配置 access_control.admin_emails / access_control.sre_emails

# 方式一：使用 Docker Compose 启动依赖服务
docker-compose up -d

# 方式二：使用本地已安装的服务
# 确保 PostgreSQL 和 Redis 已运行，更新 config.yaml 中的端口配置

# 运行数据库迁移（手动执行 SQL）
psql -h localhost -U postgres -d uniedit -f migrations/000001_create_ai_providers.up.sql
# ... 依次执行所有迁移文件

# 生成 Wire 代码
mage wire

# 构建并运行
mage dev
# 或直接运行
go build -o bin/server ./cmd/server && ./bin/server
```

### Docker Compose 服务

项目提供完整的本地开发环境：

| 服务 | 端口 | 说明 |
|------|------|------|
| PostgreSQL | 5433 | 数据库 |
| Redis | 6380 | 缓存 |
| MinIO | 9000 (API), 9001 (Console) | S3 兼容对象存储 |

### Mage 命令

```bash
mage build         # 构建服务器二进制
mage generate      # 生成所有代码 (Wire + Proto)
mage wire          # 生成 Wire 依赖注入代码
mage test          # 运行所有测试
mage testCover     # 运行测试并生成覆盖率报告
mage lint          # 运行 golangci-lint
mage vet           # 运行 go vet
mage tidy          # 运行 go mod tidy
mage clean         # 清理构建产物
mage dev           # 构建并运行开发服务器
mage all           # 完整构建流程 (tidy → generate → vet → lint → test → build)
mage ci            # CI 流程 (tidy → generate → vet → testCover)
mage install       # 安装开发工具 (wire, golangci-lint, protoc-gen-go)
mage proto         # 从 proto 规范生成 Go + Gin 接口代码
mage protoOpenAPI  # 从 proto 规范生成 OpenAPI v2 (YAML)
```

**Proto 生成：**

```bash
mage proto         # 扫描 ./api/protobuf_spec/*/*.proto，生成到 ./api/pb/*/*.go
mage protoOpenAPI  # 生成到 ./api/openapi_spec/*/*.swagger.yaml
```

## 模块说明

### 已实现模块

| 模块 | 职责 | 状态 |
|------|------|------|
| **AI** | 多提供商 AI 代理服务，智能路由、健康监控、流式响应 | ✅ 已实现 |
| **Auth** | 用户认证：OAuth、JWT、API Key 管理 | ✅ 已实现 |
| **User** | 用户管理：注册、登录、邮箱验证、密码重置 | ✅ 已实现 |
| **Billing** | 计费配额：订阅计划、用量统计、积分余额 | ✅ 已实现 |
| **Order** | 订单管理：订阅订单、充值订单、状态机 | ✅ 已实现 |
| **Payment** | 支付集成：Stripe、支付宝、微信支付 | ✅ 已实现 |
| **Git** | Git 托管：仓库管理、协作者、PR、LFS | ✅ 已实现 |
| **Collaboration** | 团队协作：团队管理、成员邀请、角色权限 | ✅ 已实现 |
| **Media** | 媒体生成：图片生成、视频生成、任务管理 | ✅ 已实现 |

### API 架构

项目采用 **Proto-first** 的接口定义方式：

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

### AI 模块

多提供商 AI 代理服务，支持智能路由、健康监控、流式响应。

| 子模块 | 职责 | 测试覆盖 |
|--------|------|----------|
| **provider** | 提供商/模型管理、健康监控、熔断器 | 59.1% |
| **routing** | 智能路由策略链（6 种策略） | 60.3% |
| **task** | 异步任务管理、外部轮询、恢复机制 | 55.6% |
| **adapter** | LLM 协议适配（OpenAI/Anthropic） | - |
| **llm** | Chat/Stream/Embed 服务 | - |
| **media** | 图片/视频生成服务 | - |
| **group** | 模型分组、选择策略、降级配置 | - |

#### 域服务要点（internal/domain/ai）

- Chat / ChatStream / Embed 三类入口，自动从请求推断能力需求（流式、工具调用、视觉、JSON 格式）并构建 `AIRoutingContext`。
- 默认策略链顺序：UserPreference (100) → HealthFilter (90) → CapabilityFilter (80) → ContextWindow (70) → CostOptimization (50，可选) → LoadBalancing (10)。策略按得分汇总后择优，并在所有候选被滤空时返回清晰错误。
- 账户池优先：可从 `accountDB.FindAvailableByProvider` 选择高优先级账号，若存在加密密钥则通过 `AICryptoPort` 解密；否则回退 Provider 主密钥。
- 健康监控：`StartHealthMonitor` 后以配置的 `HealthCheckInterval`（默认 30s）轮询 Provider，并将状态写入内存及可选 Redis 缓存，路由前会注入最新健康度。
- 失败恢复：按账户连续失败阈值（2 次降级，5 次标记不可用）与成功恢复计数驱动健康状态；成功/失败都会更新统计与用量计费（若配置了 `AIUsageRecorderPort`）。
- 成本核算：基于模型配置的 `InputCostPer1K`/`OutputCostPer1K` 计算请求成本并回填到响应的 `RoutingInfo`。

#### 路由策略

```
用户请求 → 构建 RoutingContext
    ↓
执行策略链：
  1. UserPreference (100)   → 优先用户偏好模型
  2. HealthFilter (90)      → 过滤故障提供商
  3. CapabilityFilter (80)  → 检查模型能力
  4. ContextWindow (70)     → 检查上下文大小
  5. CostOptimization (50)  → 成本优化评分
  6. LoadBalancing (10)     → 负载均衡随机
    ↓
选择最高分模型 → 返回路由结果
```

### AI 配置关键项（configs/config.example.yaml）

```yaml
ai:
  health_check_interval: 30s   # Provider 健康轮询间隔
  failure_threshold: 5         # 熔断失败阈值（账户层）
  success_threshold: 2         # 连续成功恢复阈值
  circuit_timeout: 60s         # 熔断冷却时间
  task_cleanup_interval: 5m    # 异步任务清理周期
  task_retention_period: 24h   # 任务保留时间
  max_concurrent_tasks: 100    # 并发任务上限
  embedding_cache_ttl: 24h     # Embedding 缓存时间
```

#### 支持的 AI 提供商

| 提供商 | 类型 | 能力 |
|--------|------|------|
| OpenAI | `openai` | Chat, Stream, Vision, Tools, Embedding, Image |
| Anthropic | `anthropic` | Chat, Stream, Vision |
| Google | `google` | Chat, Stream, Multimodal |
| Azure OpenAI | `azure` | Chat, Stream, Vision, Tools |
| Ollama | `ollama` | Chat, Stream (本地部署) |
| Generic | `generic` | OpenAI 兼容 API |

### Git 模块

Git 仓库托管服务，支持 Smart HTTP 协议和 LFS。

| 功能 | 说明 |
|------|------|
| **仓库管理** | 创建、更新、删除仓库，支持 code/workflow 类型 |
| **协作者** | 添加协作者，支持 read/write/admin 权限 |
| **Pull Request** | 创建、更新、合并 PR |
| **Git LFS** | 大文件存储，S3 兼容后端 |
| **Smart HTTP** | Git clone/push 协议支持 |

### Payment 模块

多支付提供商集成，支持国内外支付方式。

| 提供商 | 支持方式 |
|--------|----------|
| **Stripe** | PaymentIntent、信用卡 |
| **Alipay** | Web、H5、App、扫码 |
| **WeChat Pay** | Native、H5、App、小程序、JSAPI |

### 计划中模块

| 模块 | 职责 | 状态 |
|------|------|------|
| Workflow | 工作流仓库：搜索、Fork、执行 | 📋 计划中 |
| Render | 视频渲染服务 | 📋 计划中 |
| Publish | 发布到平台 | 📋 计划中 |

## 配置

配置支持 YAML 文件和环境变量两种方式：

```bash
# 环境变量（敏感信息推荐使用）
export UNIEDIT_DB_PASSWORD=your_password
export UNIEDIT_REDIS_PASSWORD=your_password
export UNIEDIT_JWT_SECRET=your_secret
export UNIEDIT_STORAGE_SECRET_KEY=your_key
```

详细配置项参见 [configs/config.example.yaml](configs/config.example.yaml)

## Docker 部署

```bash
# 构建镜像
docker build -f build/package/Dockerfile -t uniedit-server .

# 使用 Docker Compose 部署
cd deployments
docker-compose up -d
```

## 测试

```bash
# 运行所有测试
mage test

# 带覆盖率
mage testCover

# 指定模块测试
go test -v ./internal/domain/ai/...
go test -v ./internal/domain/auth/...
go test -v ./internal/domain/billing/...
```

### API 接口测试

启动服务后，可以测试以下接口：

```bash
# 健康检查
curl http://localhost:8080/health
# {"status":"ok","version":"v2"}

# Ping
curl http://localhost:8080/api/v1/ping
# {"message":"pong"}

# 获取套餐列表（公开接口）
curl http://localhost:8080/api/v1/billing/plans

# 获取公开仓库列表
curl http://localhost:8080/api/v1/repos/public

# 需要认证的接口（需要 Bearer Token）
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/users/me
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/ai/models
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/teams
```

### API 测试结果

| 模块 | 接口 | 状态 |
|------|------|------|
| Health | `GET /health` | ✅ |
| Ping | `GET /api/v1/ping` | ✅ |
| Auth | `GET /api/v1/auth/me` | ✅ |
| Auth | `POST /api/v1/auth/register` | ✅ |
| User | `GET /api/v1/users/me` | ✅ |
| AI | `GET /api/v1/ai/models` | ✅ |
| Billing | `GET /api/v1/billing/plans` | ✅ |
| Billing | `GET /api/v1/billing/subscription` | ✅ |
| Collaboration | `GET /api/v1/teams` | ✅ |
| Git | `GET /api/v1/repos` | ✅ |
| Order | `GET /api/v1/orders` | ✅ |
| Payment | `GET /api/v1/payments/methods` | ✅ |
| Media | `GET /api/v1/media/tasks` | ✅ |
| API Keys | `POST /api/v1/api-keys` | ✅ |
| System API Keys | `POST /api/v1/system-api-keys` | ✅ |

## 文档

- [架构设计](docs/backend-service-design.md)
- [AI 模块设计](docs/design-ai-module.md)
- [开发规范](CLAUDE.md)
- [OpenSpec 规范](openspec/)

### API 文档

项目的 API 文档由 proto 定义自动生成（OpenAPI v2 YAML）：

```bash
mage protoOpenAPI
```

输出目录：`./api/openapi_spec/*/*.swagger.yaml`

## 开发规范

详见 [CLAUDE.md](CLAUDE.md)

## License

Proprietary - All rights reserved
