# Design: Enhance Billing Quotas

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Plan Model                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ AI Quotas   │  │Storage Quota│  │ Team Quota  │              │
│  │ - Chat      │  │ - Git (MB)  │  │ - Members   │              │
│  │ - Image     │  │ - LFS (MB)  │  │             │              │
│  │ - Video     │  │             │  │             │              │
│  │ - Embedding │  │             │  │             │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Quota Checker                               │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    QuotaChecker                          │    │
│  │  - CheckAIQuota(userID, taskType, estimate)             │    │
│  │  - CheckStorageQuota(userID, additionalBytes)           │    │
│  │  - CheckTeamQuota(teamID, currentCount)                 │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│   AI Module     │ │   Git Module    │ │ Collab Module   │
│ (llm/media)     │ │ (push/upload)   │ │ (invite)        │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

## Data Model Changes

### 1. Plan Table Migration

```sql
ALTER TABLE plans ADD COLUMN monthly_chat_tokens BIGINT DEFAULT 0;
ALTER TABLE plans ADD COLUMN monthly_image_credits INTEGER DEFAULT 0;
ALTER TABLE plans ADD COLUMN monthly_video_minutes INTEGER DEFAULT 0;
ALTER TABLE plans ADD COLUMN monthly_embedding_tokens BIGINT DEFAULT 0;
ALTER TABLE plans ADD COLUMN git_storage_mb BIGINT DEFAULT -1;
ALTER TABLE plans ADD COLUMN lfs_storage_mb BIGINT DEFAULT -1;
ALTER TABLE plans ADD COLUMN max_team_members INTEGER DEFAULT 5;
```

### 2. Default Plan Values

| Plan | Chat Tokens | Image | Video(min) | Embedding | Git(MB) | LFS(MB) | Team |
|------|-------------|-------|------------|-----------|---------|---------|------|
| Free | 10K | 10 | 5 | 10K | 100 | 500 | 1 |
| Pro | 500K | 200 | 60 | 500K | 5,000 | 10,000 | 5 |
| Team | 2M | 1,000 | 300 | 2M | 50,000 | 100,000 | 50 |
| Enterprise | -1 | -1 | -1 | -1 | -1 | -1 | -1 |

### 3. Usage Tracking Keys (Redis)

```
# AI 用量（按任务类型）
quota:chat:{userID}:{YYYY-MM}       -> tokens used
quota:image:{userID}:{YYYY-MM}      -> credits used
quota:video:{userID}:{YYYY-MM}      -> minutes used
quota:embedding:{userID}:{YYYY-MM}  -> tokens used

# 存储用量（从 DB 计算，缓存）
storage:git:{userID}                -> bytes total
storage:lfs:{userID}                -> bytes total
```

## Component Design

### 1. QuotaChecker Enhancement

```go
type QuotaChecker struct {
    billingService billing.ServiceInterface
    redis          redis.UniversalClient
    gitRepo        git.Repository
    logger         *zap.Logger
}

// CheckAIQuota 检查 AI 任务配额
func (c *QuotaChecker) CheckAIQuota(ctx context.Context, userID uuid.UUID,
    taskType string, estimatedTokens int) error {

    sub, _ := c.billingService.GetSubscription(ctx, userID)
    plan := sub.Plan

    switch taskType {
    case "chat":
        if plan.MonthlyChatTokens == 0 {
            // 使用原有 MonthlyTokens 逻辑
            return c.checkLegacyTokenQuota(ctx, userID, plan, estimatedTokens)
        }
        return c.checkChatQuota(ctx, userID, plan, estimatedTokens)
    case "image":
        return c.checkImageQuota(ctx, userID, plan)
    case "video":
        return c.checkVideoQuota(ctx, userID, plan, estimatedMinutes)
    case "embedding":
        return c.checkEmbeddingQuota(ctx, userID, plan, estimatedTokens)
    }
    return nil
}

// CheckStorageQuota 检查存储配额
func (c *QuotaChecker) CheckStorageQuota(ctx context.Context, userID uuid.UUID,
    additionalBytes int64, isLFS bool) error {

    sub, _ := c.billingService.GetSubscription(ctx, userID)
    plan := sub.Plan

    // 获取当前使用量
    currentUsage := c.getUserStorageUsage(ctx, userID)

    if isLFS {
        if plan.LFSStorageMB != -1 {
            limitBytes := plan.LFSStorageMB * 1024 * 1024
            if currentUsage.LFS + additionalBytes > limitBytes {
                return ErrLFSStorageQuotaExceeded
            }
        }
    } else {
        if plan.GitStorageMB != -1 {
            limitBytes := plan.GitStorageMB * 1024 * 1024
            if currentUsage.Git + additionalBytes > limitBytes {
                return ErrGitStorageQuotaExceeded
            }
        }
    }
    return nil
}

// CheckTeamQuota 检查团队成员配额
func (c *QuotaChecker) CheckTeamQuota(ctx context.Context, teamOwnerID uuid.UUID,
    currentMemberCount int) error {

    sub, _ := c.billingService.GetSubscription(ctx, teamOwnerID)
    plan := sub.Plan

    if plan.MaxTeamMembers != -1 && currentMemberCount >= plan.MaxTeamMembers {
        return ErrTeamMemberQuotaExceeded
    }
    return nil
}
```

### 2. Storage Calculator

```go
type StorageCalculator struct {
    gitRepo git.Repository
    redis   redis.UniversalClient
    logger  *zap.Logger
}

// GetUserStorageUsage 获取用户存储使用量（带缓存）
func (c *StorageCalculator) GetUserStorageUsage(ctx context.Context,
    userID uuid.UUID) (*StorageUsage, error) {

    // 尝试从缓存获取
    cacheKey := fmt.Sprintf("storage:user:%s", userID)
    if cached := c.redis.Get(ctx, cacheKey); cached != nil {
        return parseStorageUsage(cached)
    }

    // 从数据库计算
    repos, _ := c.gitRepo.ListByOwner(ctx, userID)

    var gitTotal, lfsTotal int64
    for _, repo := range repos {
        gitTotal += repo.SizeBytes
        lfsTotal += repo.LFSSizeBytes
    }

    usage := &StorageUsage{
        GitBytes: gitTotal,
        LFSBytes: lfsTotal,
    }

    // 缓存 5 分钟
    c.redis.Set(ctx, cacheKey, usage.Serialize(), 5*time.Minute)
    return usage, nil
}

// InvalidateCache 在仓库大小变化时调用
func (c *StorageCalculator) InvalidateCache(ctx context.Context, userID uuid.UUID) {
    cacheKey := fmt.Sprintf("storage:user:%s", userID)
    c.redis.Del(ctx, cacheKey)
}
```

### 3. Integration Points

#### AI Module (llm/media)

```go
// service.go
func (s *Service) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // 配额检查
    estimatedTokens := estimateTokens(req.Messages)
    if err := s.quotaChecker.CheckAIQuota(ctx, req.UserID, "chat", estimatedTokens); err != nil {
        return nil, err
    }

    // 执行请求...

    // 记录用量（现有逻辑）
    s.usageRecorder.Record(ctx, &UsageRecord{
        TaskType: "chat",
        // ...
    })

    return resp, nil
}
```

#### Git Module

```go
// service.go
func (s *Service) ReceivePack(ctx context.Context, repoID uuid.UUID, data []byte) error {
    repo, _ := s.repo.GetByID(ctx, repoID)

    // 估算新增大小
    additionalBytes := int64(len(data))

    // 配额检查
    if err := s.quotaChecker.CheckStorageQuota(ctx, repo.OwnerID, additionalBytes, false); err != nil {
        return err
    }

    // 执行推送...

    // 更新仓库大小
    s.repo.UpdateSize(ctx, repoID)

    // 清除缓存
    s.storageCalculator.InvalidateCache(ctx, repo.OwnerID)

    return nil
}
```

#### Collaboration Module

```go
// service.go
func (s *Service) InviteMember(ctx context.Context, teamID, inviterID uuid.UUID,
    email string, role Role) error {

    team, _ := s.repo.GetTeam(ctx, teamID)
    memberCount, _ := s.repo.GetMemberCount(ctx, teamID)

    // 配额检查
    if err := s.quotaChecker.CheckTeamQuota(ctx, team.OwnerID, memberCount); err != nil {
        return err
    }

    // 创建邀请...
    return nil
}
```

## API Changes

### GET /api/v1/billing/quota

```json
{
  "ai": {
    "chat": {
      "used": 45000,
      "limit": 500000,
      "remaining": 455000
    },
    "image": {
      "used": 15,
      "limit": 200,
      "remaining": 185
    },
    "video": {
      "used_minutes": 5,
      "limit_minutes": 60,
      "remaining_minutes": 55
    },
    "embedding": {
      "used": 10000,
      "limit": 500000,
      "remaining": 490000
    }
  },
  "storage": {
    "git": {
      "used_mb": 150,
      "limit_mb": 5000,
      "remaining_mb": 4850
    },
    "lfs": {
      "used_mb": 2000,
      "limit_mb": 10000,
      "remaining_mb": 8000
    }
  },
  "team": {
    "members": {
      "current": 3,
      "limit": 5,
      "remaining": 2
    }
  },
  "reset_at": "2024-02-01T00:00:00Z"
}
```

## Migration Strategy

1. **Phase 1**: 添加新字段到 Plan 表，默认值保持兼容
2. **Phase 2**: 更新 seed 数据，设置各套餐配额
3. **Phase 3**: 实现配额检查逻辑
4. **Phase 4**: 更新 API 响应格式
5. **Phase 5**: 前端展示配额使用情况

## Testing Strategy

1. **单元测试**: QuotaChecker 各方法
2. **集成测试**: 端到端配额检查流程
3. **边界测试**: 配额刚好用完、超额场景
4. **兼容性测试**: 旧套餐（新字段为 0）仍使用原逻辑
