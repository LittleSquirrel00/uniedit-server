# Change: Add AI Module - Multi-Provider LLM Gateway

## Why

UniEdit 视频编辑器需要 AI 能力支持智能字幕、文案生成、图片/视频生成等功能。当前后端缺少统一的 AI 代理服务，需要实现：
- 多提供商管理（OpenAI、Anthropic、Google 等）
- 智能路由和负载均衡
- 异步任务管理
- 媒体生成能力

## What Changes

### 新增能力（6 个独立 Capability）

| Capability | 描述 |
|------------|------|
| **ai-provider** | AI 提供商和模型管理，数据库存储，内存缓存 |
| **ai-adapter** | LLM 适配器层，统一接口封装 OpenAI/Anthropic/Generic |
| **ai-routing** | 智能路由系统，6 种策略链 + 7 种分组选择策略 |
| **ai-task** | 异步任务管理，数据库持久化，进度跟踪 |
| **ai-media** | 媒体生成服务，图片/视频/音频生成 |
| **ai-llm-service** | LLM 服务入口，Chat/Stream/Embed API |

### 数据库变更

- 新增表：`ai_providers`, `ai_models`, `ai_groups`, `ai_tasks`
- 新增索引：按类型、能力、状态的索引

### API 端点

```
POST /api/v1/ai/chat           # 聊天补全
POST /api/v1/ai/chat/stream    # 流式聊天
POST /api/v1/ai/images/generations   # 图片生成
POST /api/v1/ai/videos/generations   # 视频生成
GET  /api/v1/ai/tasks/{id}     # 任务状态
DELETE /api/v1/ai/tasks/{id}   # 取消任务

# Admin API
GET/POST /api/v1/admin/ai/providers
GET/POST /api/v1/admin/ai/models
GET/POST /api/v1/admin/ai/groups
```

## Impact

- Affected specs: 新增 6 个能力规范
- Affected code:
  - `internal/module/ai/` - 新模块
  - `migrations/` - 数据库迁移
  - `internal/app/routes.go` - 路由注册
- Dependencies:
  - `github.com/openai/openai-go` - OpenAI 官方 SDK
  - `github.com/liushuangls/go-anthropic/v2` - Anthropic SDK
  - `github.com/sony/gobreaker` - 熔断器
