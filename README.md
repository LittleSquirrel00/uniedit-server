# UniEdit Server

UniEdit 视频编辑器的后端服务，提供用户认证、AI 代理、计费管理、工作流仓库、Git 托管等能力。

## 技术栈

| 层级 | 技术 |
|------|------|
| 语言 | Go 1.22+ |
| 框架 | Gin (HTTP) + GORM (ORM) |
| 数据库 | PostgreSQL + TimescaleDB |
| 缓存 | Redis |
| 对象存储 | Cloudflare R2 (S3 兼容) |
| 支付 | Stripe |

## 项目结构

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
│   │   └── git/             # Git 托管模块
│   └── shared/              # 共享基础设施
├── migrations/              # 数据库迁移
├── api/                     # OpenAPI 定义
├── docker/                  # Docker 配置
└── docs/                    # 设计文档
```

## 快速开始

### 环境要求

- Go 1.22+
- PostgreSQL 15+ (with TimescaleDB)
- Redis 7+

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/your-org/uniedit-server.git
cd uniedit-server

# 安装依赖
go mod download

# 复制配置文件
cp .env.example .env

# 启动依赖服务
docker-compose up -d postgres redis

# 运行数据库迁移
go run ./cmd/migrate up

# 启动服务
go run ./cmd/server
```

### 构建命令

```bash
go build -o bin/server ./cmd/server    # 编译
go run ./cmd/server                     # 运行
go test ./...                           # 测试
go test -cover ./...                    # 覆盖率
golangci-lint run                       # 代码检查
```

## 模块说明

| 模块 | 职责 | 优先级 |
|------|------|--------|
| Auth | 用户身份管理：OAuth 登录、JWT、API Key 加密存储 | P0 |
| Provider | AI 提供商管理：渠道配置、健康监控、熔断器 | P0 |
| Routing | AI API 统一代理：多渠道路由、负载均衡、故障转移 | P0 |
| Billing | 计费与配额：用量统计、配额检查、Stripe 订阅 | P0 |
| Workflow | 工作流仓库：搜索发现、Fork/Star、执行调度 | P1 |
| Registry | 模型仓库：模型元数据、Trending、评分 | P1 |
| Git | 统一版本管理：Git 协议、LFS 大文件、代码/工作流托管 | P1 |

## 文档

- [架构设计](docs/backend-service-design.md)
- [API 文档](api/openapi.yaml)

## 开发规范

详见 [CLAUDE.md](CLAUDE.md)

## License

Proprietary - All rights reserved
