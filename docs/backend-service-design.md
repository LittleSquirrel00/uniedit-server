# UniEdit 后端服务架构设计

> **版本**: v2.4 | **更新**: 2026-01-04
> **状态**: 设计阶段
> **架构**: Go 模块化单体（9 模块）
> **参考**: 客户端 platform 包架构

---

## 一、概述

### 1.1 背景与目标

UniEdit 当前为纯客户端架构（VSCode Extension），为支持商业化运营和社区生态，需要开发后端服务。

**核心业务需求**：

| 需求 | 描述 | 优先级 |
|------|------|--------|
| 用户认证 | OAuth 登录，统一身份管理 | P0 |
| API Key 托管 | 安全存储用户的 AI 服务商密钥 | P0 |
| AI 代理 | 统一 API 入口，多渠道路由 | P0 |
| 用量计费 | 记录使用量，配额管理，付费订阅 | P0 |
| 工作流仓库 | 工作流版本管理，社区分享 | P1 |
| 模型仓库 | AI 模型元数据，Trending 排行 | P1 |
| Git 托管 | 代码/工作流/项目版本管理，LFS 大文件支持 | P1 |

### 1.2 设计原则

| 原则 | 说明 |
|------|------|
| **模块化单体** | 单进程部署，模块间通过接口解耦，支持未来拆分 |
| **渐进式演进** | MVP 快速上线，按需扩展功能 |
| **参考成熟方案** | AI 代理参考 [one-api](https://github.com/songquanpeng/one-api) 设计 |
| **统一版本管理** | 代码、工作流、项目统一使用 Git + LFS 管理 |
| **成本可控** | 初期使用免费/低成本云服务，按增长扩容 |

### 1.3 技术选型

| 层级 | 选型 | 理由 |
|------|------|------|
| 语言 | Go 1.22+ | 高性能、低资源、与 One-API 统一 |
| 框架 | Gin | 成熟稳定，生态完善 |
| ORM | GORM | Go 生态首选 |
| 数据库 | PostgreSQL + TimescaleDB | 主数据 + 时序数据 |
| 缓存 | Redis | 限流、会话、配额缓存 |
| 对象存储 | Cloudflare R2 | S3 兼容，免出站费，LFS 后端 |
| CDN | Cloudflare | 媒体文件加速分发 |
| 支付 | Stripe | API 友好，全球覆盖 |

---

## 二、系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                         客户端                               │
│    VSCode Extension  │  Web Console  │  Git Client          │
└─────────────────────────────────────────────────────────────┘
                              │
                    Cloudflare (CDN + WAF)
                              │
┌─────────────────────────────────────────────────────────────┐
│                   uniedit-server (Go)                        │
├─────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────┐  │
│  │                    API Gateway                         │  │
│  │         路由 │ 认证 │ 限流 │ 日志 │ 监控              │  │
│  └───────────────────────────────────────────────────────┘  │
│                              │                               │
│  ┌─────────┬─────────┬─────────┬──────────┬──────────┐     │
│  │  Auth   │ AIProxy │ Billing │ Workflow │ Registry │     │
│  │ Module  │ Module  │ Module  │  Module  │  Module  │     │
│  └─────────┴─────────┴─────────┴──────────┴──────────┘     │
│                              │                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Git Module                           │  │
│  │         Git Protocol │ LFS API │ 仓库管理              │  │
│  └───────────────────────────────────────────────────────┘  │
│                              │                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Shared Infrastructure                     │  │
│  │      Database │ Cache │ Storage │ Crypto │ Events     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
   PostgreSQL              Redis              Cloudflare R2
   + TimescaleDB                              + CDN
```

### 2.2 模块职责

| 模块 | 职责 | 核心能力 | 优先级 |
|------|------|---------|--------|
| **Auth** | 用户身份管理 | OAuth 登录、JWT、API Key 加密存储 | P0 |
| **AIProxy** | AI API 统一代理 | 多渠道路由、负载均衡、故障转移 | P0 |
| **Billing** | 计费与配额 | 用量统计、配额检查、Stripe 订阅 | P0 |
| **Workflow** | 工作流仓库 | 搜索发现、Fork/Star、Trending | P1 |
| **Registry** | 模型仓库 | 模型元数据、Trending、评分 | P1 |
| **Git** | 统一版本管理 | Git 协议、LFS 大文件、代码/工作流/项目托管 | P1 |
| **Community** | 社区互动 | 热榜算法、关注/点赞/评论、推荐 | P2 |
| **Render** | 云端渲染 | 任务队列、分布式 Worker、进度推送 | P2 |
| **Publish** | 多平台发布 | 平台授权、一键发布、数据同步 | P2 |

### 2.3 模块依赖

```
                         ┌──────────┐
                         │   Auth   │
                         └────┬─────┘
          ┌──────────────────┼──────────────────┐
          ▼                  ▼                  ▼
     ┌────────┐        ┌──────────┐       ┌──────────┐
     │AIProxy │───────▶│ Billing  │◀──────│  Render  │
     └────────┘        └──────────┘       └────┬─────┘
                                               │
     ┌────────┐        ┌──────────┐            │
     │Registry│───────▶│Community │            │
     └────────┘        └────┬─────┘            │
                            │                  │
     ┌────────┐             ▼                  ▼
     │Workflow│───────▶┌──────────┐      ┌──────────┐
     └────────┘        │   Git    │      │ Publish  │
                       │  (+LFS)  │      └──────────┘
                       └──────────┘

规则：
• 所有模块依赖 Auth 进行鉴权
• AIProxy、Render 调用 Billing 记录用量和计费
• Workflow 使用 Git 存储工作流内容
• Render 从 Git 读取项目进行渲染
• Publish 依赖 Render 产物进行发布
• Community 聚合 Workflow、Registry 的社交数据
• 同层模块不互相依赖
```

---

## 三、模块设计

### 3.1 Auth Module（认证模块）

#### 功能需求

| 功能 | 描述 |
|------|------|
| OAuth 登录 | 支持 GitHub、Google 第三方登录 |
| JWT Token | 签发访问令牌和刷新令牌 |
| API Key 管理 | 用户存储 AI 服务商的 API Key，AES-256 加密 |
| 用户资料 | 基本信息、偏好设置 |

#### 数据模型

**User（用户）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| email | string | 邮箱，唯一 |
| name | string | 显示名称 |
| avatar_url | string | 头像 URL |
| oauth_provider | enum | github / google |
| oauth_id | string | OAuth 提供商用户 ID |
| created_at | timestamp | 创建时间 |

**APIKey（用户托管的 AI 密钥）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 所属用户 |
| provider | string | openai / anthropic / ... |
| encrypted_key | string | AES-256-GCM 加密后的密钥 |
| name | string | 用户自定义名称 |
| scopes | string[] | 权限范围：chat, image, video |
| last_used_at | timestamp | 最后使用时间 |

---

### 3.2 AIProxy Module（AI 代理模块）

> 参考 [one-api](https://github.com/songquanpeng/one-api) 的 relay 模块设计

#### 功能需求

| 功能 | 描述 |
|------|------|
| 统一 API | 对外提供 OpenAI 兼容的 API 格式 |
| 多渠道支持 | OpenAI、Anthropic、Gemini、DeepSeek 等 |
| 智能路由 | 支持负载均衡、故障转移、成本优化 |
| 用量记录 | 记录每次请求的 Token 消耗和成本 |
| 限流 | 基于用户配额的请求限制 |

#### 渠道管理

**Channel（渠道）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | string | 渠道名称 |
| type | enum | openai / anthropic / gemini / ... |
| base_url | string | API 基础 URL |
| api_key | string | 平台配置的 API Key（加密） |
| models | string[] | 支持的模型列表 |
| weight | int | 负载均衡权重 |
| status | enum | active / disabled |
| priority | int | 优先级（故障转移顺序） |

#### 路由策略

> 参考客户端 `@uniedit/platform` 包的策略链模式设计

**7 种选择策略**（可组合使用）：

| 策略 | 说明 | 适用场景 |
|------|------|----------|
| priority | 按优先级排序选择 | 默认策略，简单可控 |
| round-robin | 轮询选择 | 负载均衡 |
| weighted | 加权随机选择 | 按权重分配流量 |
| cost-optimal | 按成本优化选择 | 降低 API 成本 |
| quality-optimal | 按质量评分选择 | 追求最佳效果 |
| latency-optimal | 按延迟优化选择 | 追求响应速度 |
| capability-match | 按能力匹配选择 | 特定功能需求 |

**策略链模式**（按优先级有序执行）：

```
请求进来
    ↓
1. UserPreferenceStrategy (优先级: 100)
   ├─ 过滤被排除的提供商
   └─ 给偏好提供商/模型加分
    ↓
2. HealthFilterStrategy (优先级: 90)
   └─ 过滤不健康提供商
    ↓
3. CapabilityFilterStrategy (优先级: 80)
   └─ 过滤不支持所需能力的模型
    ↓
4. ContextWindowStrategy (优先级: 70)
   └─ 根据上下文长度评分
    ↓
5. CostOptimizationStrategy (优先级: 60)
   └─ 按成本评分
    ↓
6. LoadBalancingStrategy (优先级: 50)
   └─ 轮询或加权负载均衡
    ↓
选择累积评分最高的候选者
```

#### 模型分组管理

> 参考客户端 `GroupManager` 和 `ExecutionGroupManager`

**Group（模型分组）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | string | 分组名称 |
| models | string[] | 模型 ID 列表（按优先级排序） |
| strategy | enum | 路由策略 |
| fallback_enabled | bool | 是否启用回退 |
| fallback_triggers | string[] | 回退触发条件 |
| enabled | bool | 是否启用 |

**回退触发条件**：

| 条件 | 说明 |
|------|------|
| rate_limit | API 速率限制 |
| timeout | 请求超时 |
| server_error | 服务器错误 (5xx) |
| unavailable | 服务不可用 |
| context_length | 上下文长度超限 |

**ExecutionGroup（执行分组）** - 用于媒体生成任务路由

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| task_type | enum | 任务类型：text-to-image / image-to-video / ... |
| targets | ExecutionTarget[] | 执行目标列表 |
| strategy | enum | 路由策略 |
| enabled | bool | 是否启用 |

**ExecutionTarget（执行目标）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 目标 ID |
| type | enum | api / workflow / local |
| provider_id | string | 提供商 ID（api 类型） |
| model_id | string | 模型 ID（api 类型） |
| workflow_id | string | 工作流 ID（workflow 类型） |
| capabilities | string[] | 支持的能力 |
| estimated_latency | int | 预估延迟（ms） |
| estimated_cost | decimal | 预估成本 |
| quality_score | decimal | 质量评分 |
| enabled | bool | 是否启用 |

#### 健康监控与熔断

> 参考客户端 `ProviderRegistry` 的熔断器和健康监控

**熔断器状态**：

```
      成功请求
         │
    ┌────▼────┐
    │  CLOSED │ ←──────────────────┐
    └────┬────┘                    │
         │ 连续失败 ≥ 5 次          │ 连续成功 ≥ 2 次
         ▼                         │
    ┌─────────┐    超时 30s    ┌───┴─────┐
    │  OPEN   │───────────────▶│HALF_OPEN│
    └─────────┘                └─────────┘
         │                         │
         └─────── 请求被拒绝 ──────┘
                  (如果失败)
```

**ProviderHealth（提供商健康状态）**

| 字段 | 类型 | 说明 |
|------|------|------|
| provider_id | string | 提供商 ID |
| circuit_state | enum | closed / open / half_open |
| consecutive_failures | int | 连续失败次数 |
| last_failure_at | timestamp | 最后失败时间 |
| total_requests | bigint | 总请求数 |
| total_failures | bigint | 总失败数 |
| avg_latency_ms | int | 平均延迟 |
| updated_at | timestamp | 更新时间 |

**速率限制**：

| 配置 | 默认值 | 说明 |
|------|--------|------|
| 默认 RPM | 60 | 每分钟请求数 |
| 滑动窗口 | 60s | 限流时间窗口 |
| 自适应限流 | 开启 | 根据 API 响应动态调整 |

#### 请求流程

```
Client Request
      │
      ▼
┌─────────────┐
│  Rate Limit │ ← Redis 滑动窗口
└─────────────┘
      │
      ▼
┌─────────────┐
│ Auth Check  │ ← JWT 验证
└─────────────┘
      │
      ▼
┌─────────────┐
│ Quota Check │ ← 检查用户配额
└─────────────┘
      │
      ▼
┌─────────────┐
│ Route Select│ ← 选择最优渠道
└─────────────┘
      │
      ▼
┌─────────────┐
│ AI Provider │ → OpenAI/Anthropic/...
└─────────────┘
      │
      ▼
┌─────────────┐
│ Usage Record│ → 记录用量到 TimescaleDB
└─────────────┘
      │
      ▼
   Response
```

---

### 3.3 Billing Module（计费模块）

#### 功能需求

| 功能 | 描述 |
|------|------|
| 用量记录 | 记录每次 AI 调用的 Token 和成本 |
| 配额管理 | 月度 Token 限额、每日请求限制 |
| 订阅计划 | Free / Pro / Team / Enterprise |
| 余额系统 | 预充值余额、赠送余额 |
| 账单生成 | 月度账单、明细导出 |
| Stripe 集成 | 订阅付费、自动续费 |

#### 订阅计划

| 计划 | 价格 | 月度 Token | 每日请求 | 存储配额 | 特性 |
|------|------|-----------|---------|---------|------|
| Free | $0 | 10K | 100 | 1GB | 基础模型、限速 |
| Pro | $20/月 | 500K | 2000 | 10GB | 全部模型、LFS |
| Team | $50/月 | 2M | 10000 | 50GB | 团队协作、优先 |
| Enterprise | 定制 | 无限 | 无限 | 无限 | 私有部署、SLA |

#### 数据模型

**UsageRecord（用量记录）** - TimescaleDB 超表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 自增 ID |
| user_id | UUID | 用户 ID |
| model_id | string | 使用的模型 |
| timestamp | timestamp | 请求时间（分区键） |
| input_tokens | int | 输入 Token 数 |
| output_tokens | int | 输出 Token 数 |
| cost_usd | decimal | 成本（美元） |
| latency_ms | int | 响应延迟 |
| success | bool | 是否成功 |

**Subscription（订阅）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 用户 ID |
| plan | enum | free / pro / team / enterprise |
| status | enum | active / canceled / past_due |
| monthly_tokens | bigint | 月度 Token 配额 |
| daily_requests | int | 每日请求配额 |
| storage_quota_bytes | bigint | 存储配额（Git + LFS） |
| stripe_subscription_id | string | Stripe 订阅 ID |
| current_period_end | timestamp | 当前周期结束时间 |

---

### 3.4 Workflow Module（工作流模块）

> 参考客户端 `@uniedit/platform` 的 `WorkflowManager`

#### 功能需求

| 功能 | 描述 |
|------|------|
| 仓库索引 | 工作流仓库的元数据索引（实际存储在 Git） |
| 社区功能 | Fork、Star、下载统计 |
| 搜索发现 | 全文搜索、标签过滤、Trending |
| 版本展示 | 展示 Git 仓库的版本/标签 |
| **工作流执行** | 支持多种工作流引擎执行 |

> 注：工作流内容实际存储在 Git 仓库中，Workflow Module 主要负责索引、社区功能和执行调度

#### 工作流类型

| 类型 | 说明 | 执行方式 |
|------|------|----------|
| builtin | 内置工作流 | 服务端直接执行 |
| n8n | N8n 工作流 | 调用 N8n API |
| comfyui | ComfyUI 工作流 | 调用 ComfyUI API |
| custom | 自定义工作流 | 插件扩展 |

#### 工作流执行器模式

```
┌─────────────────────────────────────────────────────────────┐
│                   WorkflowManager                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌──────────────────────────────────────────────────────┐  │
│   │              ExecutorRegistry (注册表模式)            │  │
│   │   ┌─────────┬─────────┬─────────┬─────────┐         │  │
│   │   │BuiltIn  │  N8n    │ComfyUI  │ Custom  │         │  │
│   │   │Executor │Executor │Executor │Executor │         │  │
│   │   └─────────┴─────────┴─────────┴─────────┘         │  │
│   └──────────────────────────────────────────────────────┘  │
│                                                              │
│   核心方法：                                                 │
│   • registerExecutor(type, executor)  注册执行器            │
│   • execute(input) → WorkflowResult   执行工作流            │
│   • getExecution(id) → WorkflowResult 查询执行状态          │
│   • cancelExecution(id) → bool        取消执行              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### 数据模型

**Workflow（工作流定义）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | string | 工作流名称 |
| description | string | 描述 |
| type | enum | builtin / n8n / comfyui / custom |
| input_schema | jsonb | 输入参数 JSON Schema |
| output_schema | jsonb | 输出参数 JSON Schema |
| config | jsonb | 执行配置 |
| version | string | 版本号 |
| enabled | bool | 是否启用 |

**WorkflowRepo（工作流仓库索引）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| git_repo_id | UUID | 关联的 Git 仓库 |
| owner_id | UUID | 所有者 |
| name | string | 仓库名称 |
| slug | string | URL 友好名称 |
| description | string | 描述 |
| tags | string[] | 标签 |
| stars_count | int | Star 数 |
| forks_count | int | Fork 数 |
| downloads_count | int | 下载数 |

**WorkflowExecution（工作流执行记录）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| workflow_id | UUID | 工作流 ID |
| user_id | UUID | 用户 ID |
| status | enum | idle / running / completed / failed / cancelled |
| input | jsonb | 输入参数 |
| output | jsonb | 输出结果 |
| error | string | 错误信息 |
| started_at | timestamp | 开始时间 |
| completed_at | timestamp | 完成时间 |
| duration_ms | int | 执行时长 |

**Star / Fork** - 关联表，记录用户行为

---

### 3.5 Registry Module（模型仓库模块）

#### 功能需求

| 功能 | 描述 |
|------|------|
| 模型元数据 | 名称、能力、价格、状态 |
| Trending | 基于使用量和增长率的热度排行 |
| 搜索筛选 | 按能力、价格、提供商筛选 |
| 用户评分 | 社区评价和评论 |
| 模型对比 | 多模型能力和价格对比 |

#### 数据模型

**Model（模型）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 模型 ID（openai/gpt-4o） |
| provider | string | 提供商 |
| name | string | 显示名称 |
| capabilities | string[] | 能力：chat, vision, tools... |
| context_window | int | 上下文窗口大小 |
| pricing_input | decimal | 输入价格（$/1M tokens） |
| pricing_output | decimal | 输出价格 |
| status | enum | active / deprecated / preview |
| community_rating | decimal | 社区评分（1-5） |

#### Trending 算法

```
Score = 0.4 × 近7天使用量（归一化）
      + 0.3 × 周环比增长率
      + 0.2 × 社区评分
      + 0.1 × 新模型加成（30天内）
```

---

### 3.6 Git Module（统一版本管理）

#### 概述

Git Module 是统一的版本管理后端，支持三种仓库类型：
- **code**: 代码仓库
- **workflow**: 工作流仓库
- **project**: 视频项目仓库（使用 LFS 管理媒体文件）

#### 功能需求

| 功能 | 描述 |
|------|------|
| Git 协议 | 支持 Git Smart HTTP（clone/push/fetch） |
| **LFS 支持** | 大文件存储，预签名直传 |
| 仓库管理 | 创建、删除、设置 |
| 分支管理 | 默认分支、保护规则 |
| Pull Request | 创建、审查、合并 PR |
| 协作权限 | read / write / admin |
| 配额管理 | 仓库大小、LFS 存储配额 |

#### LFS 架构（预签名直传）

**设计要点**：
- LFS Server 只处理元数据，不传输实际文件
- 实际文件通过预签名 URL 直传 R2/CDN
- 避免服务器带宽瓶颈

```
┌─────────────────────────────────────────────────────────────┐
│                      LFS 传输流程                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Client                Server                R2 + CDN      │
│     │                     │                      │          │
│     │  1. LFS Batch       │                      │          │
│     │  POST /lfs/objects  │                      │          │
│     │ ──────────────────► │                      │          │
│     │                     │                      │          │
│     │  2. 预签名 URLs     │                      │          │
│     │ ◄────────────────── │                      │          │
│     │                     │                      │          │
│     │  3. 直连上传/下载（绕过服务器）             │          │
│     │ ═══════════════════════════════════════════►│         │
│     │                     │                      │          │
│     │  4. 确认完成        │                      │          │
│     │ ──────────────────► │  (更新元数据)        │          │
│     │                     │                      │          │
└─────────────────────────────────────────────────────────────┘

优势：
• 服务器只处理元数据 API，无带宽压力
• 实际传输走 R2 直连或 CDN 边缘节点
• 支持高并发、大文件、全球加速
```

#### 视频项目仓库结构

```
video-project.git/
├── project.jvi              # 项目配置（Git 管理）
├── timeline.json            # 时间线数据（Git 管理）
├── settings.json            # 用户设置（Git 管理）
├── media/                   # 媒体文件（Git LFS）
│   ├── video-001.mp4
│   ├── audio-001.wav
│   └── image-001.png
├── assets/                  # 素材资源（Git LFS）
│   └── ...
└── .gitattributes           # LFS 配置

# .gitattributes
media/**/*.mp4 filter=lfs diff=lfs merge=lfs -text
media/**/*.mov filter=lfs diff=lfs merge=lfs -text
media/**/*.wav filter=lfs diff=lfs merge=lfs -text
media/**/*.mp3 filter=lfs diff=lfs merge=lfs -text
media/**/*.png filter=lfs diff=lfs merge=lfs -text
media/**/*.jpg filter=lfs diff=lfs merge=lfs -text
assets/** filter=lfs diff=lfs merge=lfs -text
```

#### 数据模型

**GitRepo（Git 仓库）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| owner_id | UUID | 所有者 |
| name | string | 仓库名 |
| **repo_type** | enum | code / workflow / project |
| visibility | enum | public / private |
| default_branch | string | 默认分支 |
| size_bytes | bigint | Git 仓库大小 |
| **lfs_enabled** | bool | 是否启用 LFS |
| **lfs_size_bytes** | bigint | LFS 文件总大小 |
| storage_path | string | 存储路径 |
| created_at | timestamp | 创建时间 |
| pushed_at | timestamp | 最后推送时间 |

**LFSObject（LFS 对象）**

| 字段 | 类型 | 说明 |
|------|------|------|
| oid | string | SHA-256 哈希（主键） |
| size | bigint | 文件大小 |
| storage_key | string | R2 存储 Key |
| created_at | timestamp | 创建时间 |

**LFSRepoObject（仓库-LFS 对象关联）**

| 字段 | 类型 | 说明 |
|------|------|------|
| repo_id | UUID | 仓库 ID |
| oid | string | LFS 对象 ID |
| created_at | timestamp | 关联时间 |

> 注：LFS 对象通过 oid（内容哈希）去重，多仓库可共享同一对象

**LFSLock（文件锁定）** - 用于协作编辑

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| repo_id | UUID | 仓库 ID |
| path | string | 锁定的文件路径 |
| owner_id | UUID | 锁定者 |
| locked_at | timestamp | 锁定时间 |

**PullRequest（PR）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| repo_id | UUID | 所属仓库 |
| number | int | PR 编号 |
| title | string | 标题 |
| source_branch | string | 源分支 |
| target_branch | string | 目标分支 |
| status | enum | open / merged / closed |
| author_id | UUID | 作者 |

---

### 3.7 Render Module（渲染模块）

> 状态：P2 预留

#### 功能需求

| 功能 | 描述 |
|------|------|
| 任务提交 | 用户提交渲染任务，指定输出参数 |
| 队列管理 | 任务排队、优先级调度 |
| 分布式渲染 | 支持多 Worker 并行处理 |
| 进度同步 | SSE 实时推送渲染进度 |
| 产物管理 | 渲染结果存储、下载、过期清理 |

#### 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                     Render Module                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Client          API Server           Worker Pool          │
│     │                 │                    │                │
│     │  提交任务       │                    │                │
│     │────────────────▶│                    │                │
│     │                 │                    │                │
│     │  返回 task_id   │   入队 Redis       │                │
│     │◀────────────────│───────────────────▶│                │
│     │                 │                    │                │
│     │                 │                    │  拉取任务      │
│     │                 │                    │◀───────────    │
│     │                 │                    │                │
│     │                 │                    │  下载项目(Git) │
│     │                 │                    │───────────▶    │
│     │                 │                    │                │
│     │  SSE 进度推送   │   进度上报         │  FFmpeg 渲染   │
│     │◀────────────────│◀───────────────────│───────────▶    │
│     │                 │                    │                │
│     │                 │                    │  上传产物(R2)  │
│     │                 │                    │───────────▶    │
│     │                 │                    │                │
│     │  渲染完成通知   │   更新状态         │                │
│     │◀────────────────│◀───────────────────│                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### 数据模型

**RenderTask（渲染任务）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 用户 ID |
| project_repo_id | UUID | 项目仓库 ID |
| status | enum | pending / processing / completed / failed |
| priority | int | 优先级（付费用户更高） |
| output_format | enum | mp4 / webm / mov |
| output_quality | enum | 720p / 1080p / 4k |
| progress | int | 0-100 进度 |
| output_url | string | 渲染产物 URL |
| output_size_bytes | bigint | 产物大小 |
| expires_at | timestamp | 产物过期时间 |
| worker_id | string | 处理的 Worker |
| started_at | timestamp | 开始时间 |
| completed_at | timestamp | 完成时间 |
| error_message | string | 错误信息 |

#### 计费规则

| 计费项 | 单价 | 说明 |
|--------|------|------|
| 渲染时长 | $0.05 / 分钟 | 按输出视频时长 |
| 分辨率加成 | 720p ×0.5, 1080p ×1, 4K ×2.5 | 高分辨率更贵 |
| 优先队列 | +50% | 付费用户插队 |

---

### 3.8 Community Module（社区模块）

> 状态：P2 预留

#### 功能需求

| 功能 | 描述 |
|------|------|
| 热榜算法 | 工作流、模型、项目的 Trending 排行 |
| 社交互动 | 关注、点赞、评论、收藏 |
| 内容推荐 | 基于行为的个性化推荐 |
| 话题标签 | 内容分类、话题聚合 |
| 通知系统 | 互动消息、系统通知 |

#### 热榜算法

```
Trending Score 计算（每小时更新）：

Score = W1 × 近期互动 + W2 × 增长速度 + W3 × 质量分 + W4 × 新鲜度

其中：
├── 近期互动 = stars + forks×2 + downloads×0.5 + comments（7天内）
├── 增长速度 = (本周互动 - 上周互动) / max(上周互动, 1)
├── 质量分   = avg_rating × review_count^0.5
└── 新鲜度   = 1 / (1 + days_since_update/30)

权重（可按内容类型配置）：
├── 工作流：W1=0.4, W2=0.3, W3=0.2, W4=0.1
├── 模型：  W1=0.3, W2=0.2, W3=0.4, W4=0.1
└── 项目：  W1=0.3, W2=0.3, W3=0.2, W4=0.2
```

#### 数据模型

**Follow（关注关系）**

| 字段 | 类型 | 说明 |
|------|------|------|
| follower_id | UUID | 关注者 |
| following_id | UUID | 被关注者 |
| created_at | timestamp | 关注时间 |

**Like（点赞）**

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | UUID | 用户 ID |
| target_type | enum | workflow / model / project / comment |
| target_id | UUID | 目标 ID |
| created_at | timestamp | 点赞时间 |

**Comment（评论）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 评论者 |
| target_type | enum | workflow / model / project |
| target_id | UUID | 目标 ID |
| parent_id | UUID | 父评论（回复） |
| content | text | 评论内容 |
| likes_count | int | 点赞数 |
| created_at | timestamp | 创建时间 |

**Collection（收藏夹）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 所属用户 |
| name | string | 收藏夹名称 |
| visibility | enum | public / private |
| items_count | int | 收藏数量 |

**TrendingCache（热榜缓存）**

| 字段 | 类型 | 说明 |
|------|------|------|
| target_type | enum | workflow / model / project |
| target_id | UUID | 目标 ID |
| score | decimal | 热度分数 |
| rank | int | 排名 |
| period | enum | daily / weekly / monthly |
| updated_at | timestamp | 更新时间 |

**Notification（通知）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 接收者 |
| type | enum | follow / like / comment / system |
| actor_id | UUID | 触发者 |
| target_type | string | 目标类型 |
| target_id | UUID | 目标 ID |
| content | string | 通知内容 |
| read | bool | 是否已读 |
| created_at | timestamp | 创建时间 |

---

### 3.9 Publish Module（发布模块）

> 状态：P2 预留

#### 功能需求

| 功能 | 描述 |
|------|------|
| 平台授权 | OAuth 绑定 YouTube、TikTok、Bilibili 等 |
| 一键发布 | 渲染完成后自动/手动发布到多平台 |
| 发布配置 | 标题、描述、标签、封面、定时发布 |
| 状态同步 | 同步各平台的播放量、互动数据 |
| 发布历史 | 记录发布记录和状态 |

#### 支持平台

| 平台 | API 类型 | 功能范围 | 状态 |
|------|----------|----------|------|
| YouTube | OAuth + Data API | 上传、获取统计 | MVP |
| Bilibili | OAuth + 开放平台 | 上传、获取统计 | MVP |
| TikTok | OAuth + Content API | 上传 | MVP |
| 抖音 | 开放平台 | 上传（需企业认证） | 后续 |
| 小红书 | 开放平台 | 上传 | 后续 |

#### 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                     Publish Module                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐  │
│  │  Platform   │    │   Publish   │    │   Analytics     │  │
│  │  Connector  │    │  Scheduler  │    │   Collector     │  │
│  │             │    │             │    │                 │  │
│  │ • YouTube   │    │ • 即时发布  │    │ • 定时拉取      │  │
│  │ • Bilibili  │◀──▶│ • 定时发布  │◀──▶│ • 数据聚合      │  │
│  │ • TikTok    │    │ • 批量发布  │    │ • 趋势分析      │  │
│  │ • 抖音      │    │ • 重试机制  │    │                 │  │
│  └─────────────┘    └─────────────┘    └─────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### 数据模型

**PlatformAuth（平台授权）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 用户 ID |
| platform | enum | youtube / bilibili / tiktok / douyin |
| platform_user_id | string | 平台用户 ID |
| platform_username | string | 平台用户名 |
| access_token | string | 访问令牌（加密） |
| refresh_token | string | 刷新令牌（加密） |
| expires_at | timestamp | 过期时间 |
| scopes | string[] | 授权范围 |

**PublishTask（发布任务）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 用户 ID |
| render_task_id | UUID | 关联渲染任务 |
| platforms | string[] | 目标平台列表 |
| status | enum | pending / publishing / completed / partial_failed |
| scheduled_at | timestamp | 定时发布时间 |
| created_at | timestamp | 创建时间 |

**PublishRecord（发布记录）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| task_id | UUID | 发布任务 ID |
| platform | enum | 平台 |
| platform_video_id | string | 平台视频 ID |
| platform_url | string | 平台视频链接 |
| status | enum | pending / uploading / published / failed |
| error_message | string | 错误信息 |
| published_at | timestamp | 发布时间 |

**PublishConfig（发布配置）**

| 字段 | 类型 | 说明 |
|------|------|------|
| task_id | UUID | 任务 ID |
| platform | enum | 平台 |
| title | string | 标题 |
| description | text | 描述 |
| tags | string[] | 标签 |
| cover_url | string | 封面图 URL |
| visibility | enum | public / private / unlisted |
| category | string | 分类 |

**VideoAnalytics（视频数据）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| record_id | UUID | 发布记录 ID |
| views | bigint | 播放量 |
| likes | int | 点赞数 |
| comments | int | 评论数 |
| shares | int | 分享数 |
| watch_time_seconds | bigint | 总观看时长 |
| fetched_at | timestamp | 数据获取时间 |

---

## 四、API 设计

### 4.1 API 路由总览

```
/api/v1
├── /auth                    # 认证
│   ├── POST /login          # OAuth 登录
│   ├── POST /refresh        # 刷新 Token
│   └── POST /logout         # 登出
│
├── /users                   # 用户
│   ├── GET  /me             # 当前用户
│   └── PATCH /me            # 更新资料
│
├── /keys                    # API Key
│   ├── GET  /               # 列表
│   ├── POST /               # 创建
│   ├── DELETE /:id          # 删除
│   └── POST /:id/rotate     # 轮换
│
├── /ai                      # AI 代理
│   ├── POST /chat           # 聊天补全
│   ├── POST /chat/stream    # 流式聊天
│   └── POST /images         # 图像生成
│
├── /billing                 # 计费
│   ├── GET  /balance        # 余额
│   ├── GET  /usage          # 用量明细
│   └── POST /subscribe      # 订阅
│
├── /workflows               # 工作流
│   ├── GET  /               # 列表/搜索
│   ├── GET  /trending       # 热门
│   ├── GET  /:id            # 详情
│   ├── POST /:id/fork       # Fork
│   └── POST /:id/star       # Star
│
├── /models                  # 模型仓库
│   ├── GET  /               # 列表
│   ├── GET  /trending       # 热门
│   └── GET  /compare        # 对比
│
├── /repos                   # Git 仓库 REST API
│   ├── GET  /               # 列表
│   ├── POST /               # 创建
│   ├── GET  /:owner/:repo   # 详情
│   ├── PATCH /:owner/:repo  # 更新设置
│   ├── DELETE /:owner/:repo # 删除
│   ├── GET  /:owner/:repo/branches   # 分支列表
│   ├── GET  /:owner/:repo/commits    # 提交历史
│   ├── GET  /:owner/:repo/pulls      # PR 列表
│   └── POST /:owner/:repo/pulls      # 创建 PR
│
├── /git                     # Git Smart HTTP
│   ├── /:owner/:repo/info/refs
│   ├── /:owner/:repo/git-upload-pack
│   └── /:owner/:repo/git-receive-pack
│
├── /lfs                     # Git LFS API
│   ├── /:owner/:repo/objects/batch   # 批量上传/下载
│   ├── /:owner/:repo/locks           # 文件锁定
│   └── /:owner/:repo/locks/:id       # 解锁
│
├── /render                  # 渲染服务（P2）
│   ├── POST /tasks          # 创建渲染任务
│   ├── GET  /tasks          # 任务列表
│   ├── GET  /tasks/:id      # 任务详情
│   ├── GET  /tasks/:id/events  # SSE 进度推送
│   ├── DELETE /tasks/:id    # 取消任务
│   └── GET  /tasks/:id/download  # 下载产物
│
├── /community               # 社区（P2）
│   ├── GET  /trending/:type # 热榜（workflow/model/project）
│   ├── GET  /feed           # 个性化推荐
│   ├── POST /follow/:user_id     # 关注用户
│   ├── DELETE /follow/:user_id   # 取消关注
│   ├── POST /like/:type/:id      # 点赞
│   ├── DELETE /like/:type/:id    # 取消点赞
│   ├── /comments            # 评论 CRUD
│   ├── /collections         # 收藏夹 CRUD
│   └── /notifications       # 通知列表
│
└── /publish                 # 发布（P2）
    ├── GET  /platforms      # 已授权平台列表
    ├── POST /platforms/:name/auth    # 授权平台
    ├── DELETE /platforms/:name       # 解除授权
    ├── POST /tasks          # 创建发布任务
    ├── GET  /tasks          # 发布历史
    ├── GET  /tasks/:id      # 任务详情
    └── GET  /analytics/:record_id    # 视频数据
```

### 4.2 核心接口说明

#### 认证接口

| 接口 | 说明 |
|------|------|
| `POST /auth/login` | OAuth 登录，返回 JWT Token |
| `POST /auth/refresh` | 刷新访问令牌 |

#### AI 代理接口

| 接口 | 说明 |
|------|------|
| `POST /ai/chat` | 聊天补全，OpenAI 兼容格式 |
| `POST /ai/chat/stream` | SSE 流式响应 |

请求参数：
- `model`: 模型 ID 或 "auto"（自动路由）
- `messages`: 消息数组
- `routing`: 可选路由策略配置

响应包含：
- `model_used`: 实际使用的模型
- `usage`: Token 消耗和成本

#### LFS Batch API

| 接口 | 说明 |
|------|------|
| `POST /lfs/:owner/:repo/objects/batch` | 批量获取上传/下载 URL |

**请求示例**：
```json
{
  "operation": "download",
  "objects": [
    { "oid": "sha256:abc...", "size": 104857600 }
  ]
}
```

**响应示例**：
```json
{
  "transfer": "basic",
  "objects": [{
    "oid": "sha256:abc...",
    "size": 104857600,
    "actions": {
      "download": {
        "href": "https://cdn.example.com/lfs/abc...?sig=...",
        "expires_in": 3600
      }
    }
  }]
}
```

> 注：`href` 返回 R2/CDN 预签名 URL，客户端直连下载

---

## 五、部署架构

### 5.1 开发环境

```yaml
# Docker Compose
services:
  server:      # Go 应用
  postgres:    # PostgreSQL + TimescaleDB
  redis:       # Redis
```

### 5.2 生产环境

```
┌─────────────────────────────────────────────────────┐
│                   Cloudflare                         │
│            CDN │ WAF │ DDoS │ SSL                   │
└─────────────────────────────────────────────────────┘
                          │
┌─────────────────────────────────────────────────────┐
│              Fly.io / Railway / K8s                  │
│                                                      │
│    ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│    │  Server  │  │  Server  │  │  Server  │ (2-10) │
│    └──────────┘  └──────────┘  └──────────┘        │
└─────────────────────────────────────────────────────┘
                          │
      ┌───────────────────┼───────────────────┐
      ▼                   ▼                   ▼
 PostgreSQL            Redis           Cloudflare R2
 (Supabase/Neon)      (Upstash)        + CDN
                                       (Git + LFS)
```

### 5.3 成本估算

| 阶段 | 配置 | 月成本 |
|------|------|--------|
| MVP | 1 实例 + 免费层数据库 | $30-50 |
| 增长期 | 2-3 实例 + 付费数据库 | $100-200 |
| 规模化 | K8s 集群 + 托管服务 | $300+ |

R2 存储成本：
- 存储：$0.015/GB/月
- 出站：免费（通过 Cloudflare CDN）
- 10GB LFS 存储 ≈ $0.15/月

---

## 六、安全设计

### 6.1 认证安全

| 措施 | 说明 |
|------|------|
| OAuth 2.0 | 第三方登录，不存储密码 |
| JWT + Refresh | 短期访问令牌 + 长期刷新令牌 |
| Token 黑名单 | 登出后 Token 失效 |

### 6.2 API Key 安全

| 措施 | 说明 |
|------|------|
| AES-256-GCM | 对称加密存储 |
| 密钥轮换 | 定期更换加密密钥 |
| 不可逆 Hash | 存储 Hash 用于查找，不暴露原文 |

### 6.3 限流策略

| 层级 | 策略 |
|------|------|
| 全局 | IP 限流，防 DDoS |
| 用户 | 基于订阅计划的 RPM 限制 |
| 接口 | 敏感接口额外限制 |

### 6.4 LFS 安全

| 措施 | 说明 |
|------|------|
| 预签名 URL | 有效期限制（1 小时） |
| 权限校验 | 上传/下载前验证仓库权限 |
| 配额检查 | 上传前检查用户存储配额 |

---

## 七、实施路线图

### Phase 1: MVP（4 周）

**目标**：完成核心 AI 代理功能

- [ ] 项目初始化（Go + Gin + GORM）
- [ ] Auth Module：GitHub OAuth + JWT
- [ ] AIProxy Module：OpenAI/Anthropic 适配器
- [ ] Billing Module：用量记录 + 基础配额
- [ ] Docker Compose 部署

**交付物**：可用的 AI 代理服务

### Phase 2: 计费与社区（4 周）

**目标**：完成商业化基础

- [ ] Billing Module：Stripe 订阅集成
- [ ] Workflow Module：索引 + Fork/Star
- [ ] Registry Module：模型元数据 + 搜索
- [ ] 智能路由：Fallback + 负载均衡
- [ ] Extension 集成适配

**交付物**：可付费的完整服务

### Phase 3: Git + LFS（4 周）

**目标**：统一版本管理

- [ ] Git Module：Smart HTTP 协议
- [ ] LFS 支持：Batch API + 预签名直传
- [ ] 仓库类型：code / workflow / project
- [ ] 分支管理 + PR
- [ ] 协作者权限
- [ ] 存储配额管理

**交付物**：完整的 Git + LFS 托管服务

### Phase 4: 社区功能（4 周）

**目标**：完善社区互动

- [ ] Community Module：热榜算法
- [ ] 关注 / 点赞 / 评论
- [ ] 收藏夹功能
- [ ] 通知系统
- [ ] 个性化推荐

**交付物**：完整的社区互动功能

### Phase 5: 云端渲染（4 周）

**目标**：支持云端视频渲染

- [ ] Render Module：任务队列
- [ ] Worker 调度系统
- [ ] FFmpeg 渲染引擎集成
- [ ] 进度推送（SSE）
- [ ] 产物存储与过期清理
- [ ] 渲染计费

**交付物**：可用的云端渲染服务

### Phase 6: 多平台发布（4 周）

**目标**：支持视频分发

- [ ] Publish Module：平台授权
- [ ] YouTube / Bilibili / TikTok 集成
- [ ] 一键发布 / 定时发布
- [ ] 视频数据同步
- [ ] 发布历史管理

**交付物**：多平台一键发布能力

### Phase 7: 优化与扩展（持续）

- [ ] Trending 算法优化
- [ ] 评价系统
- [ ] Web Console
- [ ] 文件锁定（LFS Lock）
- [ ] 更多平台支持（抖音、小红书）
- [ ] 实时协作（未来）

---

## 八、项目结构

```
uniedit-server/
├── cmd/server/              # 程序入口
├── internal/
│   ├── app/                 # 应用组装、路由
│   ├── module/              # 业务模块
│   │   ├── auth/
│   │   ├── aiproxy/
│   │   ├── billing/
│   │   ├── workflow/
│   │   ├── registry/
│   │   ├── git/             # Git + LFS
│   │   │   ├── handler.go   # HTTP Handler
│   │   │   ├── service.go   # 业务逻辑
│   │   │   ├── repository.go
│   │   │   ├── protocol.go  # Git 协议
│   │   │   └── lfs/         # LFS 子模块
│   │   │       ├── batch.go
│   │   │       ├── storage.go
│   │   │       └── lock.go
│   │   ├── community/       # 社区模块（P2）
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── trending.go  # 热榜算法
│   │   │   └── notification.go
│   │   ├── render/          # 渲染模块（P2）
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── queue.go     # 任务队列
│   │   │   └── worker/      # Worker 引擎
│   │   └── publish/         # 发布模块（P2）
│   │       ├── handler.go
│   │       ├── service.go
│   │       ├── platforms/   # 平台适配器
│   │       │   ├── youtube.go
│   │       │   ├── bilibili.go
│   │       │   └── tiktok.go
│   │       └── analytics.go
│   └── shared/              # 共享基础设施
│       ├── config/
│       ├── database/
│       ├── cache/
│       ├── storage/         # R2/S3 客户端
│       ├── middleware/
│       └── crypto/
├── migrations/              # 数据库迁移
├── docker/                  # Docker 配置
└── api/                     # OpenAPI 定义
```

---

## 九、关键决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| 架构模式 | 模块化单体 | 初期快速迭代，降低运维复杂度 |
| 技术栈 | Go + Gin | 高性能、低资源，与 One-API 统一 |
| AI 代理设计 | 参考 One-API | 成熟方案，减少重复开发 |
| **项目存储** | **Git + LFS** | 统一版本管理，复用 Git 基础设施 |
| **LFS 传输** | **预签名直传** | 避免服务器带宽瓶颈，支持 CDN 加速 |
| 对象存储 | Cloudflare R2 | S3 兼容，免出站费用 |
| 时序数据 | TimescaleDB | 基于 PG，降低运维成本 |
| 未来拆分 | Git Module 优先 | I/O 密集型模块优先独立 |
| **路由策略** | **策略链模式** | 参考客户端 platform 包，可扩展的多策略组合 |
| **分组管理** | **Group + ExecutionGroup** | 模型分组 + 执行分组，灵活路由 |
| **健康监控** | **熔断器 + 速率限制** | 故障隔离，保护上游服务 |

---

## 附录 A：客户端 Platform 包架构参考

> 后端设计参考客户端 `@uniedit/platform` 包的架构，确保前后端架构一致性

### A.1 包结构总览

```
packages/platform/src/
├── core/                # 核心抽象层
│   ├── selection-strategy.ts   # 7 种选择策略
│   ├── router.ts              # 路由基类（策略链）
│   ├── circuit-breaker.ts     # 熔断器
│   ├── rate-limiter.ts        # 速率限制
│   └── health-monitor.ts      # 健康监控
│
├── config/              # 配置管理
│   └── config-manager.ts      # 三层配置合并
│
├── provider/            # 提供商管理
│   ├── provider-registry.ts   # 提供商注册表
│   ├── group-manager.ts       # 模型分组
│   └── execution-group-manager.ts  # 执行分组
│
├── llm/                 # LLM 层
│   ├── adapter/               # 提供商适配器
│   │   ├── openai.ts
│   │   ├── anthropic.ts
│   │   └── gemini.ts
│   └── routing/               # LLM 智能路由
│       └── llm-routing-manager.ts
│
├── media/               # 媒体生成
│   └── routing/               # 媒体路由
│       └── media-routing-manager.ts
│
├── workflow/            # 工作流引擎
│   └── workflow-manager.ts
│
└── types/               # 类型定义
```

### A.2 核心设计模式

| 模式 | 位置 | 用途 |
|------|------|------|
| **策略模式** | `core/selection-strategy.ts` | 7 种可插拔的选择策略 |
| **策略链** | `llm/routing/` | 多策略有序执行和评分 |
| **注册表模式** | `provider/provider-registry.ts` | 统一的组件注册和查询 |
| **工厂模式** | `core/selection-strategy.ts` | `SelectionStrategyFactory` |
| **模板方法** | `core/router.ts` | `BaseRoutingManager` 定义路由流程 |
| **熔断器模式** | `core/circuit-breaker.ts` | 故障隔离和自恢复 |
| **观察者模式** | `config/config-manager.ts` | 配置变更监听 |
| **适配器模式** | `llm/adapter/` | 提供商 API 适配 |

### A.3 配置管理（三层优先级）

```
优先级从高到低：
┌──────────────────────────────┐
│ 工作区配置                    │ ← .uniedit/config.json
│ (Workspace Config)           │
├──────────────────────────────┤
│ 用户配置                      │ ← VS Code globalState
│ (User Config)                │
├──────────────────────────────┤
│ 内置预设                      │ ← builtin-presets
│ (Builtin Presets)            │
└──────────────────────────────┘

合并后输出：
• providers: Map<string, Provider>
• models: Map<string, Model>
• groups: Map<string, Group>
• executionGroups: Map<string, ExecutionGroup>
• workflows: Map<string, Workflow>
• ...
```

### A.4 模块依赖关系

```
types/
  ↑
  ├── core/          (基础构建块)
  │    ├── selection-strategy.ts
  │    └── router.ts
  │
  ├── config/        (配置管理)
  │    └─→ types
  │
  ├── provider/      (提供商管理)
  │    ├─→ config
  │    ├─→ core
  │    └─→ GroupManager, ExecutionGroupManager
  │
  ├── llm/routing/   (LLM 智能路由)
  │    ├─→ core (继承 BaseRoutingManager)
  │    ├─→ provider
  │    └─→ config
  │
  ├── media/routing/ (媒体路由)
  │    ├─→ core
  │    ├─→ provider
  │    └─→ config
  │
  └── workflow/      (工作流引擎)
       ├─→ config
       ├─→ provider
       └─→ types
```

### A.5 典型路由流程

**LLM Chat 请求**：

```
用户请求 (chat)
      │
[LLMRoutingManager.routeChat()]
      │
[策略链执行]
  1. UserPreferenceStrategy (100) → 过滤/加分偏好
  2. HealthFilterStrategy (90)    → 过滤不健康提供商
  3. CapabilityFilterStrategy (80)→ 过滤不支持能力的模型
  4. ContextWindowStrategy (70)   → 按上下文长度评分
  5. CostOptimizationStrategy (60)→ 按成本评分
  6. LoadBalancingStrategy (50)   → 负载均衡评分
      │
[选择累积评分最高的候选者]
      │
LLMRoutingResult {
  providerId: 'anthropic',
  modelId: 'claude-3-5-sonnet',
  score: 85.5,
  reason: '按策略选择...'
}
```

**媒体生成请求**：

```
生成图片请求 (text-to-image)
      │
[ExecutionGroupManager.route('image_generation')]
      │
[获取执行组 → 遍历执行目标 → 验证可用性]
      │
[按策略选择最佳目标]
  - priority: 第一个可用
  - cost-optimal: 最低成本
  - quality-optimal: 最高质量
  - latency-optimal: 最低延迟
      │
ExecutionRoutingResult {
  target: { type: 'api', providerId: 'openai', modelId: 'dall-e-3' },
  reason: '按成本优化策略选择'
}
```

### A.6 SOLID 原则应用

| 原则 | 应用示例 |
|------|----------|
| **S - 单一职责** | GroupManager 只负责分组，ExecutionGroupManager 只负责执行分组 |
| **O - 开闭原则** | ISelectionStrategy 接口支持添加新策略而无需修改现有代码 |
| **L - 里氏替换** | 所有策略实现均实现 ISelectionStrategy 接口，可互换 |
| **I - 接口隔离** | 选择策略接口只包含必需的 select 方法 |
| **D - 依赖倒置** | 依赖抽象接口（IRouter、ISelectionStrategy）而非具体实现 |

### A.7 后端实现建议

| 客户端模块 | 后端对应 | 实现建议 |
|------------|----------|----------|
| `ConfigManager` | `shared/config/` | Redis 缓存 + DB 持久化 |
| `ProviderRegistry` | `module/aiproxy/registry.go` | 内存状态 + Redis 共享 |
| `GroupManager` | `module/aiproxy/group.go` | DB 存储分组配置 |
| `LLMRoutingManager` | `module/aiproxy/router.go` | 策略链模式，可扩展 |
| `CircuitBreaker` | `shared/middleware/circuit.go` | Redis 分布式状态 |
| `WorkflowManager` | `module/workflow/executor.go` | 执行器注册表模式 |

**Go 策略链实现示例**：

```go
// 策略接口
type RoutingStrategy interface {
    Priority() int
    Apply(ctx *RoutingContext, candidates []Model) []ScoredModel
}

// 策略链
type StrategyChain struct {
    strategies []RoutingStrategy
}

func (c *StrategyChain) Route(ctx *RoutingContext, candidates []Model) *RoutingResult {
    // 按优先级排序
    sort.Slice(c.strategies, func(i, j int) bool {
        return c.strategies[i].Priority() > c.strategies[j].Priority()
    })

    scored := make([]ScoredModel, len(candidates))
    for i, m := range candidates {
        scored[i] = ScoredModel{Model: m, Score: 0}
    }

    // 依次应用策略
    for _, strategy := range c.strategies {
        scored = strategy.Apply(ctx, scored)
        if len(scored) == 0 {
            return nil // 全部过滤
        }
    }

    // 返回最高分
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].Score > scored[j].Score
    })

    return &RoutingResult{
        Model:  scored[0].Model,
        Score:  scored[0].Score,
        Reason: "按策略链选择",
    }
}
```
