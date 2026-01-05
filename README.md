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
├── cmd/server/              # 程序入口
├── internal/
│   ├── app/                 # 应用组装、路由
│   ├── module/              # 业务模块
│   │   └── ai/              # AI 代理模块
│   │       ├── adapter/     # LLM 适配器 (OpenAI/Anthropic)
│   │       ├── provider/    # 提供商管理 (Registry/Health)
│   │       ├── routing/     # 智能路由 (Strategy Chain)
│   │       ├── group/       # 模型分组
│   │       ├── task/        # 异步任务管理
│   │       ├── llm/         # LLM 服务
│   │       ├── media/       # 媒体生成服务
│   │       ├── cache/       # Embedding 缓存
│   │       └── handler/     # HTTP 处理器
│   └── shared/              # 共享基础设施
│       ├── config/          # 配置管理 (Viper)
│       ├── database/        # 数据库连接 (GORM)
│       ├── cache/           # Redis 缓存
│       ├── middleware/      # HTTP 中间件
│       └── errors/          # 错误定义
├── configs/                 # 配置文件模板
├── build/package/           # Docker 构建
├── deployments/             # 部署配置
├── migrations/              # 数据库迁移
├── scripts/                 # 脚本工具
├── docs/                    # 设计文档
└── openspec/                # OpenSpec 规范
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
```

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/uniedit/server.git
cd uniedit-server

# 复制配置文件
cp configs/config.example.yaml config.yaml
# 编辑 config.yaml 配置数据库和 Redis

# 使用 Docker Compose 启动依赖服务
cd deployments && docker-compose up -d postgres redis && cd ..

# 生成 Wire 代码
mage wire

# 构建并运行
mage dev
```

### Mage 命令

```bash
mage build      # 构建服务器二进制
mage wire       # 生成 Wire 依赖注入代码
mage test       # 运行所有测试
mage testCover  # 运行测试并生成覆盖率报告
mage lint       # 运行 golangci-lint
mage vet        # 运行 go vet
mage tidy       # 运行 go mod tidy
mage clean      # 清理构建产物
mage dev        # 构建并运行开发服务器
mage all        # 完整构建流程 (tidy → wire → vet → lint → test → build)
mage install    # 安装开发工具
```

## 模块说明

### AI 模块（已实现）

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

#### 支持的 AI 提供商

| 提供商 | 类型 | 能力 |
|--------|------|------|
| OpenAI | `openai` | Chat, Stream, Vision, Tools, Embedding, Image |
| Anthropic | `anthropic` | Chat, Stream, Vision |
| Google | `google` | Chat, Stream, Multimodal |
| Azure OpenAI | `azure` | Chat, Stream, Vision, Tools |
| Ollama | `ollama` | Chat, Stream (本地部署) |
| Generic | `generic` | OpenAI 兼容 API |

### 其他模块（计划中）

| 模块 | 职责 | 状态 |
|------|------|------|
| Auth | 用户认证：OAuth、JWT、API Key | 📋 计划中 |
| Billing | 计费配额：用量统计、订阅管理 | 📋 计划中 |
| Workflow | 工作流仓库：搜索、Fork、执行 | 📋 计划中 |

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
go test -v ./internal/module/ai/routing/...
go test -v ./internal/module/ai/provider/...
go test -v ./internal/module/ai/task/...
```

当前核心模块测试覆盖率：
- provider: 59.1%
- routing: 60.3%
- task: 55.6%

## 文档

- [架构设计](docs/backend-service-design.md)
- [AI 模块设计](docs/design-ai-module.md)
- [开发规范](CLAUDE.md)
- [OpenSpec 规范](openspec/)

## 开发规范

详见 [CLAUDE.md](CLAUDE.md)

## License

Proprietary - All rights reserved
