# Proposal: Enhance Billing Quotas

## Summary

增强计费系统，支持按任务类型差异化计费、Git 存储空间限制、团队成员配额管理。

## Motivation

当前计费系统存在以下不足：

1. **LLM 用量** - 已实现 Token 配额，但所有任务类型共用同一配额
2. **Media 用量** - `TaskType` 字段已预留，但无独立配额
3. **Git 存储空间** - 完全缺失，无法限制用户存储使用
4. **团队成员数量** - 用 `MaxAPIKeys` 代替，不准确

## Goals

1. 扩展 Plan 模型，支持按任务类型的独立配额
2. 添加 Git/LFS 存储空间限制
3. 添加团队成员数量限额
4. 实现各维度的用量追踪和配额检查

## Non-Goals

- 不改变现有 Token 计费逻辑（保持兼容）
- 不涉及支付流程变更
- 不改变 Stripe 集成

## Design

### 1. Plan 模型扩展

```go
type Plan struct {
    // 现有字段保持不变...
    MonthlyTokens int64 // 总 Token 限额（保持兼容）
    DailyRequests int   // 日请求限额（保持兼容）

    // 新增：按任务类型配额
    MonthlyChatTokens     int64 // Chat Token 限额 (-1=无限, 0=使用 MonthlyTokens)
    MonthlyImageCredits   int   // 图片生成次数 (-1=无限)
    MonthlyVideoMinutes   int   // 视频生成时长(分钟) (-1=无限)
    MonthlyEmbeddingTokens int64 // Embedding Token 限额 (-1=无限)

    // 新增：存储配额
    GitStorageMB  int64 // Git 存储空间(MB) (-1=无限)
    LFSStorageMB  int64 // LFS 存储空间(MB) (-1=无限)

    // 新增：团队配额
    MaxTeamMembers int // 团队成员上限 (-1=无限)
}
```

### 2. 配额检查增强

- Chat/Completion 请求检查 `MonthlyChatTokens`
- Image 请求检查 `MonthlyImageCredits`
- Video 请求检查 `MonthlyVideoMinutes`
- Embedding 请求检查 `MonthlyEmbeddingTokens`
- Git push 检查 `GitStorageMB + LFSStorageMB`
- 团队邀请检查 `MaxTeamMembers`

### 3. 用量记录扩展

现有 `UsageRecord.TaskType` 已支持：
- `chat` - 消耗 `MonthlyChatTokens`
- `image` - 消耗 `MonthlyImageCredits`
- `video` - 消耗 `MonthlyVideoMinutes`
- `embedding` - 消耗 `MonthlyEmbeddingTokens`

存储用量通过 `GitRepo.SizeBytes` 和 `GitRepo.LFSSizeBytes` 追踪。

## Spec Deltas

- `billing-quotas`: 按任务类型的独立配额
- `storage-quotas`: Git/LFS 存储限制
- `team-quotas`: 团队成员配额

## Risks & Mitigations

| 风险 | 缓解措施 |
|-----|---------|
| 现有订阅不兼容 | 新字段默认 0，表示使用原有 MonthlyTokens |
| 配额检查性能 | Redis 缓存各维度用量 |
| 存储计算开销 | 异步计算存储使用量 |

## Alternatives Considered

1. **分离 Plan 表** - 复杂度高，放弃
2. **动态配额 JSON** - 类型安全差，放弃
3. **每种资源独立订阅** - 用户体验差，放弃
