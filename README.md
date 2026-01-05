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

## 项目结构

```
uniedit-server/
├── cmd/server/              # 程序入口
├── internal/
│   ├── app/                 # 应用组装、路由
│   ├── module/              # 业务模块
│   │   └── ai/              # AI 代理模块
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

### 传统构建命令

```bash
go build -o bin/server ./cmd/server    # 编译
go run ./cmd/server                     # 运行
go test ./...                           # 测试
go test -cover ./...                    # 覆盖率
golangci-lint run                       # 代码检查
```

## 模块说明

| 模块 | 职责 | 状态 |
|------|------|------|
| AI | AI 代理服务：多渠道路由、健康监控、熔断器、流式响应 | ✅ 已实现 |
| Auth | 用户身份管理：OAuth 登录、JWT、API Key 加密存储 | 📋 计划中 |
| Billing | 计费与配额：用量统计、配额检查、订阅管理 | 📋 计划中 |
| Workflow | 工作流仓库：搜索发现、Fork/Star、执行调度 | 📋 计划中 |

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

## 文档

- [架构设计](docs/backend-service-design.md)
- [AI 模块设计](docs/design-ai-module.md)
- [开发规范](CLAUDE.md)

## 开发规范

详见 [CLAUDE.md](CLAUDE.md)

## License

Proprietary - All rights reserved
