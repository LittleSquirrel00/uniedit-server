# AI Module Design Document

## Context

UniEdit Server 需要集成多个 AI 提供商（OpenAI、Anthropic、Google 等）为视频编辑器提供智能功能。核心挑战：
- 多提供商统一接口
- 智能路由和故障转移
- 异步任务管理（媒体生成耗时长）
- 成本控制和监控

## Goals / Non-Goals

**Goals:**
- 统一的 LLM API 接口，对前端透明
- 智能路由：健康检查、能力匹配、成本优化
- 可扩展的适配器架构，易于添加新提供商
- 异步任务持久化，支持服务重启恢复

**Non-Goals:**
- 用户自定义 Provider（仅平台配置）
- 复杂的对话历史管理（客户端负责）
- 实时协作功能

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Handlers                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ ChatHandler │  │MediaHandler │  │ TaskHandler │          │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘          │
├─────────┼────────────────┼────────────────┼─────────────────┤
│         ▼                ▼                ▼                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ LLMService  │  │MediaService │  │ TaskManager │          │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘          │
├─────────┼────────────────┼────────────────┼─────────────────┤
│         └────────────────┼────────────────┘                 │
│                          ▼                                  │
│                 ┌─────────────────┐                         │
│                 │ RoutingManager  │                         │
│                 │  (策略链路由)    │                         │
│                 └────────┬────────┘                         │
├──────────────────────────┼──────────────────────────────────┤
│                          ▼                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              ProviderRegistry                        │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │   │
│  │  │ Provider │  │ Provider │  │ Provider │  ...      │   │
│  │  │ (OpenAI) │  │(Anthropic)│ │ (Google) │           │   │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘           │   │
│  └───────┼─────────────┼─────────────┼─────────────────┘   │
├──────────┼─────────────┼─────────────┼──────────────────────┤
│          ▼             ▼             ▼                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              AdapterRegistry                         │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │   │
│  │  │ OpenAI   │  │Anthropic │  │ Generic  │  ...      │   │
│  │  │ Adapter  │  │ Adapter  │  │ Adapter  │           │   │
│  │  └──────────┘  └──────────┘  └──────────┘           │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Decisions

### D1: 使用官方 SDK 组合而非完整 Gateway

**Decision:** 组合 `openai/openai-go` + `liushuangls/go-anthropic` + 自研路由层

**Alternatives:**
- one-api: 功能完整但需独立部署，架构复杂
- BricksLLM: 企业级但过重
- LangChainGo: 抽象过多，性能开销

**Rationale:** 保持架构可控，利用官方 SDK 稳定性，按需实现路由策略

### D2: 配置存储在数据库，启动时加载到内存

**Decision:** Provider/Model/Group 配置存数据库，启动时全量加载到内存

**Rationale:**
- 支持运行时动态更新（通过 Admin API）
- 避免每次请求查询数据库
- 内存缓存 + 定时刷新机制

### D3: 6 种路由策略 + 策略链模式

**Decision:** 使用策略链模式，按优先级执行过滤和评分

| 策略 | 优先级 | 作用 |
|------|--------|------|
| UserPreference | 100 | 用户偏好过滤 |
| HealthFilter | 90 | 健康状态过滤 |
| CapabilityFilter | 80 | 能力匹配过滤 |
| ContextWindow | 70 | 上下文窗口过滤 |
| CostOptimization | 50 | 成本评分 |
| LoadBalancing | 10 | 负载均衡随机 |

**Rationale:** 可插拔、可扩展、易测试

### D4: 异步任务持久化到数据库

**Decision:** 所有任务（特别是媒体生成）持久化到 `ai_tasks` 表

**Rationale:**
- 服务重启后可恢复未完成任务
- 支持任务历史查询
- 外部任务 ID 映射（如 Runway）

### D5: Embedding 缓存

**Decision:** 对 Embedding 结果进行 Redis 缓存

**Rationale:**
- 相同文本向量固定，命中率高
- 成本节省显著（Embedding 调用量大）
- 实现简单，无副作用

## Data Model

```sql
-- 提供商
ai_providers (id, name, type, base_url, api_key, enabled, weight, priority, rate_limit, options)

-- 模型
ai_models (id, provider_id, name, capabilities[], context_window, max_output_tokens, input_cost_per_1k, output_cost_per_1k, enabled, options)

-- 分组
ai_groups (id, name, task_type, models[], strategy, fallback, required_capabilities[], enabled)

-- 任务
ai_tasks (id, user_id, type, status, progress, input, output, error, external_task_id, provider_id, model_id, created_at, updated_at, completed_at)
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Provider API 变更 | 适配器需更新 | 使用官方 SDK，关注版本变更 |
| 内存缓存不一致 | 路由错误 | 定时刷新 + Admin API 强制刷新 |
| 异步任务堆积 | 系统负载 | 并发池限制 + 优先级队列 |
| API Key 泄露 | 安全风险 | AES-256 加密存储，日志脱敏 |

## Migration Plan

1. **Phase 1**: 数据库迁移 + 基础类型
2. **Phase 2**: 适配器层（OpenAI 优先）
3. **Phase 3**: Provider 管理 + 健康监控
4. **Phase 4**: 路由系统
5. **Phase 5**: 任务系统
6. **Phase 6**: 媒体生成
7. **Phase 7**: HTTP API
8. **Phase 8**: 测试

## Open Questions

- [ ] 是否需要支持 Azure OpenAI？（当前设计支持 Generic 适配器）
- [ ] Embedding 缓存 TTL 设置多少合适？（建议 7 天）
- [ ] 任务历史保留多久？（建议 30 天）
