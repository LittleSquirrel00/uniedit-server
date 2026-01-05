# P2 生态模块设计

> **版本**: v1.0 | **更新**: 2026-01-04
> **范围**: Community + Render + Publish
> **目标**: 构建创作者生态系统

---

## 一、模块概览

### 1.1 模块架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        P2 生态架构                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                     Community                            │   │
│   │         热榜 │ 关注 │ 点赞 │ 评论 │ 推荐 │ 通知         │   │
│   └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              │ 聚合社交数据                      │
│                              ▼                                   │
│          ┌───────────────────┼───────────────────┐              │
│          │                   │                   │              │
│   ┌──────▼─────┐      ┌──────▼─────┐      ┌──────▼─────┐       │
│   │  Workflow  │      │  Registry  │      │    Git     │       │
│   │   (P1)     │      │   (P1)     │      │   (P1)     │       │
│   └────────────┘      └────────────┘      └────────────┘       │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                      Render                              │   │
│   │           任务队列 │ Worker 调度 │ FFmpeg 渲染           │   │
│   └──────────────────────────┬──────────────────────────────┘   │
│                              │                                   │
│                              │ 渲染产物                          │
│                              ▼                                   │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                      Publish                             │   │
│   │          YouTube │ Bilibili │ TikTok │ 抖音             │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 模块职责

| 模块 | 核心职责 | 关键能力 |
|------|----------|----------|
| **Community** | 社区互动与推荐 | 热榜算法、关注/点赞/评论、个性化推荐、通知 |
| **Render** | 云端视频渲染 | 任务队列、Worker 调度、进度推送、产物管理 |
| **Publish** | 多平台发布 | 平台授权、一键发布、数据同步、发布历史 |

### 1.3 用户流程

```
┌────────────────────────────────────────────────────────────────┐
│                      创作者工作流程                              │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│   1. 创作                2. 渲染                3. 发布         │
│   ┌─────────┐           ┌─────────┐           ┌─────────┐      │
│   │  编辑   │──────────▶│  提交   │──────────▶│  选择   │      │
│   │  视频   │           │  渲染   │           │  平台   │      │
│   └─────────┘           └────┬────┘           └────┬────┘      │
│                              │                     │           │
│                         ┌────▼────┐           ┌────▼────┐      │
│                         │ Worker  │           │ 一键    │      │
│                         │  处理   │           │  发布   │      │
│                         └────┬────┘           └────┬────┘      │
│                              │                     │           │
│                         ┌────▼────┐           ┌────▼────┐      │
│                         │  进度   │           │  数据   │      │
│                         │  推送   │           │  同步   │      │
│                         └────┬────┘           └─────────┘      │
│                              │                                  │
│                         ┌────▼────┐                            │
│                         │  产物   │                            │
│                         │  下载   │                            │
│                         └─────────┘                            │
│                                                                 │
│   4. 社区互动                                                   │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  Star │ Fork │ 评论 │ 关注 │ 热榜 │ 推荐 │ 通知        │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## 二、Community Module

### 2.1 目录结构

```
internal/module/community/
├── handler.go              # HTTP Handler
├── service.go              # 业务逻辑
├── repository.go           # 数据访问
├── model.go                # 数据模型
├── dto.go                  # 请求/响应 DTO
├── trending.go             # 热榜算法
├── recommendation.go       # 推荐算法
├── notification.go         # 通知服务
├── feed.go                 # Feed 流
└── errors.go               # 模块错误
```

### 2.2 数据模型

```go
// model.go

type TargetType string

const (
    TargetTypeWorkflow TargetType = "workflow"
    TargetTypeModel    TargetType = "model"
    TargetTypeProject  TargetType = "project"
    TargetTypeComment  TargetType = "comment"
)

type NotificationType string

const (
    NotificationTypeFollow  NotificationType = "follow"
    NotificationTypeLike    NotificationType = "like"
    NotificationTypeComment NotificationType = "comment"
    NotificationTypeMention NotificationType = "mention"
    NotificationTypeFork    NotificationType = "fork"
    NotificationTypeStar    NotificationType = "star"
    NotificationTypeSystem  NotificationType = "system"
)

// Follow 关注关系
type Follow struct {
    FollowerID  uuid.UUID `gorm:"type:uuid;primaryKey"`
    FollowingID uuid.UUID `gorm:"type:uuid;primaryKey"`
    CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (Follow) TableName() string { return "follows" }

// Like 点赞
type Like struct {
    UserID     uuid.UUID  `gorm:"type:uuid;primaryKey"`
    TargetType TargetType `gorm:"primaryKey"`
    TargetID   uuid.UUID  `gorm:"type:uuid;primaryKey"`
    CreatedAt  time.Time  `gorm:"autoCreateTime"`
}

func (Like) TableName() string { return "likes" }

// Comment 评论
type Comment struct {
    ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID      uuid.UUID  `gorm:"type:uuid;not null;index"`
    TargetType  TargetType `gorm:"not null;index"`
    TargetID    uuid.UUID  `gorm:"type:uuid;not null;index"`
    ParentID    *uuid.UUID `gorm:"type:uuid;index"` // 回复的父评论
    Content     string     `gorm:"type:text;not null"`
    LikesCount  int        `gorm:"default:0"`
    RepliesCount int       `gorm:"default:0"`
    CreatedAt   time.Time  `gorm:"autoCreateTime"`
    UpdatedAt   time.Time  `gorm:"autoUpdateTime"`

    // Relations
    User     *auth.User `gorm:"foreignKey:UserID"`
    Parent   *Comment   `gorm:"foreignKey:ParentID"`
    Replies  []Comment  `gorm:"foreignKey:ParentID"`
}

func (Comment) TableName() string { return "comments" }

// Collection 收藏夹
type Collection struct {
    ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID      uuid.UUID  `gorm:"type:uuid;not null;index"`
    Name        string     `gorm:"not null"`
    Description string
    Visibility  string     `gorm:"default:private"` // public, private
    ItemsCount  int        `gorm:"default:0"`
    CreatedAt   time.Time  `gorm:"autoCreateTime"`
    UpdatedAt   time.Time  `gorm:"autoUpdateTime"`

    // Relations
    Items []CollectionItem `gorm:"foreignKey:CollectionID"`
}

func (Collection) TableName() string { return "collections" }

// CollectionItem 收藏项
type CollectionItem struct {
    CollectionID uuid.UUID  `gorm:"type:uuid;primaryKey"`
    TargetType   TargetType `gorm:"primaryKey"`
    TargetID     uuid.UUID  `gorm:"type:uuid;primaryKey"`
    Note         string     // 收藏备注
    CreatedAt    time.Time  `gorm:"autoCreateTime"`
}

func (CollectionItem) TableName() string { return "collection_items" }

// TrendingCache 热榜缓存
type TrendingCache struct {
    TargetType TargetType `gorm:"primaryKey"`
    TargetID   uuid.UUID  `gorm:"type:uuid;primaryKey"`
    Period     string     `gorm:"primaryKey"` // daily, weekly, monthly
    Score      float64    `gorm:"type:decimal(10,4);not null"`
    Rank       int        `gorm:"not null"`
    UpdatedAt  time.Time  `gorm:"autoUpdateTime"`
}

func (TrendingCache) TableName() string { return "trending_cache" }

// Notification 通知
type Notification struct {
    ID         uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID     uuid.UUID        `gorm:"type:uuid;not null;index"`
    Type       NotificationType `gorm:"not null"`
    ActorID    uuid.UUID        `gorm:"type:uuid"` // 触发者
    TargetType TargetType       `gorm:"not null"`
    TargetID   uuid.UUID        `gorm:"type:uuid;not null"`
    Content    string           // 通知内容
    Read       bool             `gorm:"default:false;index"`
    CreatedAt  time.Time        `gorm:"autoCreateTime"`

    // Relations
    Actor *auth.User `gorm:"foreignKey:ActorID"`
}

func (Notification) TableName() string { return "notifications" }

// UserStats 用户统计
type UserStats struct {
    UserID         uuid.UUID `gorm:"type:uuid;primaryKey"`
    FollowersCount int       `gorm:"default:0"`
    FollowingCount int       `gorm:"default:0"`
    WorkflowsCount int       `gorm:"default:0"`
    ProjectsCount  int       `gorm:"default:0"`
    StarsReceived  int       `gorm:"default:0"`
    UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (UserStats) TableName() string { return "user_stats" }
```

### 2.3 热榜算法

```go
// trending.go

type TrendingPeriod string

const (
    TrendingDaily   TrendingPeriod = "daily"
    TrendingWeekly  TrendingPeriod = "weekly"
    TrendingMonthly TrendingPeriod = "monthly"
)

// TrendingWeights 热榜计算权重（按内容类型配置）
type TrendingWeights struct {
    RecentInteraction float64 // 近期互动
    GrowthRate        float64 // 增长速度
    QualityScore      float64 // 质量分
    Freshness         float64 // 新鲜度
}

var TrendingWeightsByType = map[TargetType]TrendingWeights{
    TargetTypeWorkflow: {
        RecentInteraction: 0.4,
        GrowthRate:        0.3,
        QualityScore:      0.2,
        Freshness:         0.1,
    },
    TargetTypeModel: {
        RecentInteraction: 0.3,
        GrowthRate:        0.2,
        QualityScore:      0.4,
        Freshness:         0.1,
    },
    TargetTypeProject: {
        RecentInteraction: 0.3,
        GrowthRate:        0.3,
        QualityScore:      0.2,
        Freshness:         0.2,
    },
}

type TrendingCalculator struct {
    repo   Repository
    cache  cache.Cache
    logger *zap.Logger
}

func NewTrendingCalculator(repo Repository, cache cache.Cache, logger *zap.Logger) *TrendingCalculator {
    return &TrendingCalculator{
        repo:   repo,
        cache:  cache,
        logger: logger,
    }
}

// CalculateAll 计算所有类型的热榜
func (c *TrendingCalculator) CalculateAll(ctx context.Context) error {
    types := []TargetType{TargetTypeWorkflow, TargetTypeModel, TargetTypeProject}
    periods := []TrendingPeriod{TrendingDaily, TrendingWeekly, TrendingMonthly}

    for _, targetType := range types {
        for _, period := range periods {
            if err := c.Calculate(ctx, targetType, period); err != nil {
                c.logger.Error("failed to calculate trending",
                    zap.String("type", string(targetType)),
                    zap.String("period", string(period)),
                    zap.Error(err),
                )
            }
        }
    }

    return nil
}

// Calculate 计算指定类型和周期的热榜
func (c *TrendingCalculator) Calculate(ctx context.Context, targetType TargetType, period TrendingPeriod) error {
    weights := TrendingWeightsByType[targetType]

    // 获取时间范围
    now := time.Now()
    var startTime time.Time
    switch period {
    case TrendingDaily:
        startTime = now.AddDate(0, 0, -1)
    case TrendingWeekly:
        startTime = now.AddDate(0, 0, -7)
    case TrendingMonthly:
        startTime = now.AddDate(0, -1, 0)
    }

    // 获取所有目标
    targets, err := c.repo.GetTargetsByType(ctx, targetType)
    if err != nil {
        return err
    }

    // 计算每个目标的分数
    scores := make([]TrendingScore, 0, len(targets))

    for _, target := range targets {
        score := c.calculateScore(ctx, target, weights, startTime, now)
        scores = append(scores, TrendingScore{
            TargetType: targetType,
            TargetID:   target.ID,
            Score:      score,
        })
    }

    // 按分数排序
    sort.Slice(scores, func(i, j int) bool {
        return scores[i].Score > scores[j].Score
    })

    // 保存到缓存
    for i, s := range scores {
        s.Rank = i + 1
        if err := c.repo.SaveTrendingCache(ctx, &TrendingCache{
            TargetType: s.TargetType,
            TargetID:   s.TargetID,
            Period:     string(period),
            Score:      s.Score,
            Rank:       s.Rank,
        }); err != nil {
            c.logger.Error("failed to save trending cache", zap.Error(err))
        }
    }

    return nil
}

func (c *TrendingCalculator) calculateScore(ctx context.Context, target Target, weights TrendingWeights, startTime, endTime time.Time) float64 {
    // 1. 近期互动 = stars + forks×2 + downloads×0.5 + comments
    interaction := c.getInteraction(ctx, target, startTime, endTime)
    normalizedInteraction := math.Log1p(interaction) / 10 // 对数归一化

    // 2. 增长速度 = (本期互动 - 上期互动) / max(上期互动, 1)
    previousStart := startTime.Add(startTime.Sub(endTime))
    previousInteraction := c.getInteraction(ctx, target, previousStart, startTime)
    var growthRate float64
    if previousInteraction > 0 {
        growthRate = (interaction - previousInteraction) / previousInteraction
    } else if interaction > 0 {
        growthRate = 1.0
    }
    normalizedGrowth := math.Min(growthRate, 2.0) / 2 // 限制在 0-1

    // 3. 质量分 = avg_rating × sqrt(review_count)
    rating, count := c.getRating(ctx, target)
    qualityScore := rating * math.Sqrt(float64(count)) / 10 // 归一化

    // 4. 新鲜度 = 1 / (1 + days_since_update/30)
    daysSinceUpdate := endTime.Sub(target.UpdatedAt).Hours() / 24
    freshness := 1.0 / (1.0 + daysSinceUpdate/30)

    // 综合评分
    score := weights.RecentInteraction*normalizedInteraction +
        weights.GrowthRate*normalizedGrowth +
        weights.QualityScore*qualityScore +
        weights.Freshness*freshness

    return score * 100 // 放大到 0-100 范围
}

type TrendingScore struct {
    TargetType TargetType
    TargetID   uuid.UUID
    Score      float64
    Rank       int
}
```

### 2.4 推荐算法

```go
// recommendation.go

type RecommendationEngine struct {
    repo   Repository
    cache  cache.Cache
    logger *zap.Logger
}

// RecommendationType 推荐类型
type RecommendationType string

const (
    RecommendSimilar    RecommendationType = "similar"    // 相似内容
    RecommendPopular    RecommendationType = "popular"    // 热门内容
    RecommendFollowing  RecommendationType = "following"  // 关注的人的内容
    RecommendPersonal   RecommendationType = "personal"   // 个性化推荐
)

// GetRecommendations 获取推荐内容
func (e *RecommendationEngine) GetRecommendations(ctx context.Context, userID uuid.UUID, targetType TargetType, limit int) ([]*Recommendation, error) {
    var recommendations []*Recommendation

    // 1. 从关注的人获取内容 (40%)
    followingRecs := e.getFollowingRecommendations(ctx, userID, targetType, limit*4/10)
    recommendations = append(recommendations, followingRecs...)

    // 2. 基于用户兴趣的相似内容 (30%)
    similarRecs := e.getSimilarRecommendations(ctx, userID, targetType, limit*3/10)
    recommendations = append(recommendations, similarRecs...)

    // 3. 热门内容补充 (30%)
    remaining := limit - len(recommendations)
    if remaining > 0 {
        popularRecs := e.getPopularRecommendations(ctx, targetType, remaining)
        recommendations = append(recommendations, popularRecs...)
    }

    // 去重和排序
    recommendations = e.deduplicate(recommendations)
    recommendations = e.shuffle(recommendations)

    if len(recommendations) > limit {
        recommendations = recommendations[:limit]
    }

    return recommendations, nil
}

func (e *RecommendationEngine) getFollowingRecommendations(ctx context.Context, userID uuid.UUID, targetType TargetType, limit int) []*Recommendation {
    // 获取用户关注的人
    following, _ := e.repo.GetFollowing(ctx, userID)

    if len(following) == 0 {
        return nil
    }

    // 获取他们的最新内容
    followingIDs := make([]uuid.UUID, len(following))
    for i, f := range following {
        followingIDs[i] = f.FollowingID
    }

    items, _ := e.repo.GetRecentByUsers(ctx, targetType, followingIDs, limit)

    recs := make([]*Recommendation, len(items))
    for i, item := range items {
        recs[i] = &Recommendation{
            TargetType: targetType,
            TargetID:   item.ID,
            Score:      1.0,
            Reason:     RecommendFollowing,
        }
    }

    return recs
}

func (e *RecommendationEngine) getSimilarRecommendations(ctx context.Context, userID uuid.UUID, targetType TargetType, limit int) []*Recommendation {
    // 获取用户的互动历史（点赞、收藏）
    interactions, _ := e.repo.GetUserInteractions(ctx, userID, targetType)

    if len(interactions) == 0 {
        return nil
    }

    // 提取标签/特征
    tags := make(map[string]int)
    for _, item := range interactions {
        for _, tag := range item.Tags {
            tags[tag]++
        }
    }

    // 获取相似内容
    topTags := e.getTopTags(tags, 5)
    items, _ := e.repo.GetByTags(ctx, targetType, topTags, limit)

    // 过滤已交互的内容
    interactedIDs := make(map[uuid.UUID]bool)
    for _, item := range interactions {
        interactedIDs[item.ID] = true
    }

    recs := make([]*Recommendation, 0)
    for _, item := range items {
        if !interactedIDs[item.ID] {
            recs = append(recs, &Recommendation{
                TargetType: targetType,
                TargetID:   item.ID,
                Score:      0.8,
                Reason:     RecommendSimilar,
            })
        }
    }

    return recs
}

func (e *RecommendationEngine) getPopularRecommendations(ctx context.Context, targetType TargetType, limit int) []*Recommendation {
    // 从热榜获取
    trending, _ := e.repo.GetTrending(ctx, targetType, TrendingWeekly, limit)

    recs := make([]*Recommendation, len(trending))
    for i, t := range trending {
        recs[i] = &Recommendation{
            TargetType: targetType,
            TargetID:   t.TargetID,
            Score:      0.6,
            Reason:     RecommendPopular,
        }
    }

    return recs
}

type Recommendation struct {
    TargetType TargetType
    TargetID   uuid.UUID
    Score      float64
    Reason     RecommendationType
}
```

### 2.5 通知服务

```go
// notification.go

type NotificationService struct {
    repo   Repository
    cache  cache.Cache
    logger *zap.Logger
}

func NewNotificationService(repo Repository, cache cache.Cache, logger *zap.Logger) *NotificationService {
    return &NotificationService{
        repo:   repo,
        cache:  cache,
        logger: logger,
    }
}

// Send 发送通知
func (s *NotificationService) Send(ctx context.Context, notification *Notification) error {
    // 保存到数据库
    if err := s.repo.CreateNotification(ctx, notification); err != nil {
        return err
    }

    // 增加未读计数
    key := fmt.Sprintf("unread:%s", notification.UserID)
    s.cache.Incr(ctx, key)

    // TODO: 实时推送（WebSocket/SSE）

    return nil
}

// SendBatch 批量发送通知
func (s *NotificationService) SendBatch(ctx context.Context, userIDs []uuid.UUID, notification *Notification) error {
    for _, userID := range userIDs {
        notif := *notification
        notif.ID = uuid.New()
        notif.UserID = userID

        if err := s.Send(ctx, &notif); err != nil {
            s.logger.Error("failed to send notification",
                zap.String("user_id", userID.String()),
                zap.Error(err),
            )
        }
    }
    return nil
}

// NotifyFollowers 通知所有粉丝
func (s *NotificationService) NotifyFollowers(ctx context.Context, actorID uuid.UUID, targetType TargetType, targetID uuid.UUID, notifType NotificationType) error {
    followers, err := s.repo.GetFollowers(ctx, actorID)
    if err != nil {
        return err
    }

    followerIDs := make([]uuid.UUID, len(followers))
    for i, f := range followers {
        followerIDs[i] = f.FollowerID
    }

    return s.SendBatch(ctx, followerIDs, &Notification{
        Type:       notifType,
        ActorID:    actorID,
        TargetType: targetType,
        TargetID:   targetID,
    })
}

// GetUnreadCount 获取未读数
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
    key := fmt.Sprintf("unread:%s", userID)
    count, err := s.cache.Get(ctx, key).Int()
    if err == redis.Nil {
        // 从数据库获取
        count, _ = s.repo.GetUnreadNotificationCount(ctx, userID)
        s.cache.Set(ctx, key, count, 24*time.Hour)
    }
    return count, nil
}

// MarkAsRead 标记已读
func (s *NotificationService) MarkAsRead(ctx context.Context, userID uuid.UUID, notificationID uuid.UUID) error {
    if err := s.repo.MarkNotificationRead(ctx, notificationID); err != nil {
        return err
    }

    // 减少未读计数
    key := fmt.Sprintf("unread:%s", userID)
    s.cache.Decr(ctx, key)

    return nil
}

// MarkAllAsRead 全部标记已读
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
    if err := s.repo.MarkAllNotificationsRead(ctx, userID); err != nil {
        return err
    }

    key := fmt.Sprintf("unread:%s", userID)
    s.cache.Set(ctx, key, 0, 24*time.Hour)

    return nil
}
```

### 2.6 核心接口

```go
// service.go

type Service interface {
    // 关注
    Follow(ctx context.Context, userID, targetUserID uuid.UUID) error
    Unfollow(ctx context.Context, userID, targetUserID uuid.UUID) error
    GetFollowers(ctx context.Context, userID uuid.UUID, page, pageSize int) (*FollowList, error)
    GetFollowing(ctx context.Context, userID uuid.UUID, page, pageSize int) (*FollowList, error)
    IsFollowing(ctx context.Context, userID, targetUserID uuid.UUID) (bool, error)

    // 点赞
    Like(ctx context.Context, userID uuid.UUID, targetType TargetType, targetID uuid.UUID) error
    Unlike(ctx context.Context, userID uuid.UUID, targetType TargetType, targetID uuid.UUID) error
    IsLiked(ctx context.Context, userID uuid.UUID, targetType TargetType, targetID uuid.UUID) (bool, error)
    GetLikesCount(ctx context.Context, targetType TargetType, targetID uuid.UUID) (int, error)

    // 评论
    CreateComment(ctx context.Context, userID uuid.UUID, req *CreateCommentRequest) (*Comment, error)
    UpdateComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, content string) (*Comment, error)
    DeleteComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
    GetComments(ctx context.Context, targetType TargetType, targetID uuid.UUID, page, pageSize int) (*CommentList, error)
    GetReplies(ctx context.Context, commentID uuid.UUID, page, pageSize int) (*CommentList, error)

    // 收藏夹
    CreateCollection(ctx context.Context, userID uuid.UUID, req *CreateCollectionRequest) (*Collection, error)
    UpdateCollection(ctx context.Context, userID uuid.UUID, collectionID uuid.UUID, req *UpdateCollectionRequest) (*Collection, error)
    DeleteCollection(ctx context.Context, userID uuid.UUID, collectionID uuid.UUID) error
    GetCollections(ctx context.Context, userID uuid.UUID) ([]*Collection, error)
    AddToCollection(ctx context.Context, userID uuid.UUID, collectionID uuid.UUID, targetType TargetType, targetID uuid.UUID) error
    RemoveFromCollection(ctx context.Context, userID uuid.UUID, collectionID uuid.UUID, targetType TargetType, targetID uuid.UUID) error

    // 热榜
    GetTrending(ctx context.Context, targetType TargetType, period TrendingPeriod, limit int) ([]*TrendingItem, error)
    RefreshTrending(ctx context.Context) error

    // 推荐
    GetFeed(ctx context.Context, userID uuid.UUID, page, pageSize int) (*FeedList, error)
    GetRecommendations(ctx context.Context, userID uuid.UUID, targetType TargetType, limit int) ([]*Recommendation, error)

    // 通知
    GetNotifications(ctx context.Context, userID uuid.UUID, page, pageSize int) (*NotificationList, error)
    GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
    MarkAsRead(ctx context.Context, userID uuid.UUID, notificationID uuid.UUID) error
    MarkAllAsRead(ctx context.Context, userID uuid.UUID) error

    // 用户统计
    GetUserStats(ctx context.Context, userID uuid.UUID) (*UserStats, error)
}

type CreateCommentRequest struct {
    TargetType TargetType `json:"target_type" binding:"required"`
    TargetID   uuid.UUID  `json:"target_id" binding:"required"`
    ParentID   *uuid.UUID `json:"parent_id"` // 回复时填写
    Content    string     `json:"content" binding:"required,max=2000"`
}

type TrendingItem struct {
    Rank       int        `json:"rank"`
    TargetType TargetType `json:"target_type"`
    TargetID   uuid.UUID  `json:"target_id"`
    Score      float64    `json:"score"`
    Target     interface{} `json:"target"` // 具体内容
}
```

### 2.7 API 接口

```yaml
# Follow API

POST /api/v1/users/{user_id}/follow:
  description: 关注用户

DELETE /api/v1/users/{user_id}/follow:
  description: 取消关注

GET /api/v1/users/{user_id}/followers:
  description: 粉丝列表
  query:
    page: int
    page_size: int

GET /api/v1/users/{user_id}/following:
  description: 关注列表

# Like API

POST /api/v1/like/{type}/{id}:
  description: 点赞
  path:
    type: workflow | model | project | comment
    id: uuid

DELETE /api/v1/like/{type}/{id}:
  description: 取消点赞

# Comment API

GET /api/v1/comments:
  description: 获取评论
  query:
    target_type: string
    target_id: uuid
    page: int

POST /api/v1/comments:
  description: 创建评论
  request:
    target_type: string
    target_id: uuid
    parent_id: uuid (optional)
    content: string

DELETE /api/v1/comments/{id}:
  description: 删除评论

# Collection API

GET /api/v1/collections:
  description: 我的收藏夹

POST /api/v1/collections:
  description: 创建收藏夹
  request:
    name: string
    description: string
    visibility: public | private

POST /api/v1/collections/{id}/items:
  description: 添加到收藏夹
  request:
    target_type: string
    target_id: uuid

DELETE /api/v1/collections/{id}/items/{target_type}/{target_id}:
  description: 从收藏夹移除

# Trending API

GET /api/v1/trending/{type}:
  description: 热榜
  path:
    type: workflow | model | project
  query:
    period: daily | weekly | monthly
    limit: int

# Feed API

GET /api/v1/feed:
  description: 个人化 Feed
  query:
    page: int
    page_size: int

GET /api/v1/recommendations/{type}:
  description: 推荐内容
  path:
    type: workflow | model | project
  query:
    limit: int

# Notification API

GET /api/v1/notifications:
  description: 通知列表
  query:
    page: int
    unread_only: bool

GET /api/v1/notifications/unread-count:
  description: 未读数

POST /api/v1/notifications/{id}/read:
  description: 标记已读

POST /api/v1/notifications/read-all:
  description: 全部标记已读

# User Stats API

GET /api/v1/users/{user_id}/stats:
  description: 用户统计
  response:
    followers_count: int
    following_count: int
    workflows_count: int
    projects_count: int
    stars_received: int
```

---

## 三、Render Module

### 3.1 目录结构

```
internal/module/render/
├── handler.go              # HTTP Handler
├── service.go              # 业务逻辑
├── repository.go           # 数据访问
├── model.go                # 数据模型
├── dto.go                  # 请求/响应 DTO
├── queue.go                # 任务队列
├── scheduler.go            # 任务调度器
├── sse.go                  # SSE 进度推送
├── worker/                 # Worker 实现
│   ├── interface.go        # Worker 接口
│   ├── pool.go             # Worker 池
│   ├── ffmpeg.go           # FFmpeg 渲染引擎
│   └── handler.go          # 任务处理
└── errors.go               # 模块错误
```

### 3.2 数据模型

```go
// model.go

type TaskStatus string

const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusProcessing TaskStatus = "processing"
    TaskStatusCompleted  TaskStatus = "completed"
    TaskStatusFailed     TaskStatus = "failed"
    TaskStatusCancelled  TaskStatus = "cancelled"
)

type OutputFormat string

const (
    OutputFormatMP4  OutputFormat = "mp4"
    OutputFormatWebM OutputFormat = "webm"
    OutputFormatMOV  OutputFormat = "mov"
)

type OutputQuality string

const (
    OutputQuality720p  OutputQuality = "720p"
    OutputQuality1080p OutputQuality = "1080p"
    OutputQuality4K    OutputQuality = "4k"
)

// RenderTask 渲染任务
type RenderTask struct {
    ID              uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID          uuid.UUID     `gorm:"type:uuid;not null;index"`
    ProjectRepoID   uuid.UUID     `gorm:"type:uuid;not null"` // 项目仓库 ID
    ProjectCommit   string        `gorm:"not null"`           // Git commit SHA
    Status          TaskStatus    `gorm:"not null;default:pending;index"`
    Priority        int           `gorm:"default:0"`          // 优先级（付费用户更高）
    OutputFormat    OutputFormat  `gorm:"not null;default:mp4"`
    OutputQuality   OutputQuality `gorm:"not null;default:1080p"`
    OutputFPS       int           `gorm:"default:30"`

    // 进度
    Progress        int           `gorm:"default:0"`          // 0-100
    CurrentStep     string        // 当前步骤描述

    // 产物
    OutputURL       string        // 渲染产物 URL
    OutputSizeBytes int64         // 产物大小
    ThumbnailURL    string        // 缩略图 URL
    Duration        float64       // 视频时长（秒）
    ExpiresAt       *time.Time    // 产物过期时间

    // Worker 信息
    WorkerID        string        // 处理的 Worker
    StartedAt       *time.Time
    CompletedAt     *time.Time
    ErrorMessage    string

    // 计费
    EstimatedCost   float64       `gorm:"type:decimal(10,4)"` // 预估成本
    ActualCost      float64       `gorm:"type:decimal(10,4)"` // 实际成本

    CreatedAt       time.Time     `gorm:"autoCreateTime"`
    UpdatedAt       time.Time     `gorm:"autoUpdateTime"`

    // Relations
    User            *auth.User    `gorm:"foreignKey:UserID"`
}

func (RenderTask) TableName() string { return "render_tasks" }

// RenderPreset 渲染预设
type RenderPreset struct {
    ID          uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID      *uuid.UUID    `gorm:"type:uuid;index"` // nil 表示系统预设
    Name        string        `gorm:"not null"`
    Format      OutputFormat  `gorm:"not null"`
    Quality     OutputQuality `gorm:"not null"`
    FPS         int           `gorm:"not null"`
    Codec       string        `gorm:"not null"` // h264, h265, vp9
    Bitrate     string        // 如 "8M"
    AudioCodec  string        `gorm:"default:aac"`
    AudioBitrate string       `gorm:"default:192k"`
    IsDefault   bool          `gorm:"default:false"`
    CreatedAt   time.Time     `gorm:"autoCreateTime"`
}

func (RenderPreset) TableName() string { return "render_presets" }
```

### 3.3 任务队列

```go
// queue.go

type Queue interface {
    // 入队
    Push(ctx context.Context, task *RenderTask) error

    // 出队（阻塞）
    Pop(ctx context.Context) (*RenderTask, error)

    // 获取队列长度
    Length(ctx context.Context) (int, error)

    // 获取任务位置
    Position(ctx context.Context, taskID uuid.UUID) (int, error)

    // 移除任务
    Remove(ctx context.Context, taskID uuid.UUID) error
}

// RedisQueue Redis 实现的优先级队列
type RedisQueue struct {
    client *redis.Client
    key    string
    logger *zap.Logger
}

func NewRedisQueue(client *redis.Client, key string, logger *zap.Logger) *RedisQueue {
    return &RedisQueue{
        client: client,
        key:    key,
        logger: logger,
    }
}

func (q *RedisQueue) Push(ctx context.Context, task *RenderTask) error {
    // 使用优先级作为分数（负数，因为分数越小越优先）
    score := float64(-task.Priority*1000000) + float64(task.CreatedAt.Unix())

    data, err := json.Marshal(task)
    if err != nil {
        return err
    }

    return q.client.ZAdd(ctx, q.key, redis.Z{
        Score:  score,
        Member: string(data),
    }).Err()
}

func (q *RedisQueue) Pop(ctx context.Context) (*RenderTask, error) {
    // 阻塞获取分数最低（优先级最高）的任务
    result, err := q.client.BZPopMin(ctx, 0, q.key).Result()
    if err != nil {
        return nil, err
    }

    var task RenderTask
    if err := json.Unmarshal([]byte(result.Member.(string)), &task); err != nil {
        return nil, err
    }

    return &task, nil
}

func (q *RedisQueue) Position(ctx context.Context, taskID uuid.UUID) (int, error) {
    // 获取所有任务
    members, err := q.client.ZRange(ctx, q.key, 0, -1).Result()
    if err != nil {
        return -1, err
    }

    for i, m := range members {
        var task RenderTask
        if err := json.Unmarshal([]byte(m), &task); err != nil {
            continue
        }
        if task.ID == taskID {
            return i + 1, nil // 1-based position
        }
    }

    return -1, nil // Not found
}
```

### 3.4 Worker 设计

```go
// worker/interface.go

type Worker interface {
    // ID 返回 Worker ID
    ID() string

    // Start 启动 Worker
    Start(ctx context.Context) error

    // Stop 停止 Worker
    Stop() error

    // Status 获取状态
    Status() WorkerStatus
}

type WorkerStatus struct {
    ID          string    `json:"id"`
    State       string    `json:"state"` // idle, busy, stopping
    CurrentTask *uuid.UUID `json:"current_task"`
    TasksCompleted int64  `json:"tasks_completed"`
    LastHeartbeat time.Time `json:"last_heartbeat"`
}

// worker/pool.go

type Pool struct {
    workers    []Worker
    queue      Queue
    service    Service
    logger     *zap.Logger
    mu         sync.RWMutex
    cancelFunc context.CancelFunc
}

func NewPool(workerCount int, queue Queue, service Service, logger *zap.Logger) *Pool {
    pool := &Pool{
        workers: make([]Worker, workerCount),
        queue:   queue,
        service: service,
        logger:  logger,
    }

    for i := 0; i < workerCount; i++ {
        pool.workers[i] = NewFFmpegWorker(
            fmt.Sprintf("worker-%d", i),
            queue,
            service,
            logger,
        )
    }

    return pool
}

func (p *Pool) Start(ctx context.Context) error {
    ctx, p.cancelFunc = context.WithCancel(ctx)

    for _, worker := range p.workers {
        go func(w Worker) {
            if err := w.Start(ctx); err != nil {
                p.logger.Error("worker error", zap.String("id", w.ID()), zap.Error(err))
            }
        }(worker)
    }

    p.logger.Info("worker pool started", zap.Int("workers", len(p.workers)))
    return nil
}

func (p *Pool) Stop() error {
    if p.cancelFunc != nil {
        p.cancelFunc()
    }

    for _, worker := range p.workers {
        worker.Stop()
    }

    p.logger.Info("worker pool stopped")
    return nil
}

// worker/ffmpeg.go

type FFmpegWorker struct {
    id      string
    queue   Queue
    service Service
    logger  *zap.Logger
    state   string
    current *uuid.UUID
    mu      sync.RWMutex
}

func NewFFmpegWorker(id string, queue Queue, service Service, logger *zap.Logger) *FFmpegWorker {
    return &FFmpegWorker{
        id:      id,
        queue:   queue,
        service: service,
        logger:  logger,
        state:   "idle",
    }
}

func (w *FFmpegWorker) Start(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        default:
        }

        // 获取任务
        task, err := w.queue.Pop(ctx)
        if err != nil {
            if ctx.Err() != nil {
                return nil
            }
            w.logger.Error("failed to pop task", zap.Error(err))
            time.Sleep(time.Second)
            continue
        }

        w.setCurrentTask(task.ID)

        // 处理任务
        if err := w.processTask(ctx, task); err != nil {
            w.logger.Error("task failed",
                zap.String("task_id", task.ID.String()),
                zap.Error(err),
            )
            w.service.UpdateTaskStatus(ctx, task.ID, TaskStatusFailed, err.Error())
        }

        w.setCurrentTask(uuid.Nil)
    }
}

func (w *FFmpegWorker) processTask(ctx context.Context, task *RenderTask) error {
    w.logger.Info("processing task", zap.String("task_id", task.ID.String()))

    // 1. 更新状态为处理中
    w.service.UpdateTaskStatus(ctx, task.ID, TaskStatusProcessing, "")
    w.service.UpdateProgress(ctx, task.ID, 0, "Preparing...")

    // 2. 下载项目文件（从 Git）
    projectPath, err := w.downloadProject(ctx, task)
    if err != nil {
        return fmt.Errorf("download project: %w", err)
    }
    defer os.RemoveAll(projectPath)

    w.service.UpdateProgress(ctx, task.ID, 10, "Project downloaded")

    // 3. 解析项目配置
    config, err := w.parseProjectConfig(projectPath)
    if err != nil {
        return fmt.Errorf("parse config: %w", err)
    }

    w.service.UpdateProgress(ctx, task.ID, 15, "Config parsed")

    // 4. 构建 FFmpeg 命令
    outputPath := filepath.Join(os.TempDir(), task.ID.String()+"."+string(task.OutputFormat))
    cmd := w.buildFFmpegCommand(config, task, outputPath)

    // 5. 执行渲染（带进度回调）
    if err := w.executeFFmpeg(ctx, task, cmd); err != nil {
        return fmt.Errorf("ffmpeg: %w", err)
    }

    w.service.UpdateProgress(ctx, task.ID, 95, "Uploading...")

    // 6. 上传产物到 R2
    outputURL, size, err := w.uploadOutput(ctx, task, outputPath)
    if err != nil {
        return fmt.Errorf("upload: %w", err)
    }

    // 7. 生成缩略图
    thumbnailURL, _ := w.generateThumbnail(ctx, task, outputPath)

    // 8. 更新任务完成
    w.service.CompleteTask(ctx, task.ID, &CompleteTaskResult{
        OutputURL:       outputURL,
        OutputSizeBytes: size,
        ThumbnailURL:    thumbnailURL,
        Duration:        config.Duration,
    })

    w.logger.Info("task completed", zap.String("task_id", task.ID.String()))
    return nil
}

func (w *FFmpegWorker) executeFFmpeg(ctx context.Context, task *RenderTask, cmd *exec.Cmd) error {
    // 创建管道读取进度
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return err
    }

    if err := cmd.Start(); err != nil {
        return err
    }

    // 解析 FFmpeg 进度输出
    go w.parseProgress(ctx, task, stderr)

    // 等待完成
    return cmd.Wait()
}

func (w *FFmpegWorker) parseProgress(ctx context.Context, task *RenderTask, stderr io.Reader) {
    scanner := bufio.NewScanner(stderr)
    timeRegex := regexp.MustCompile(`time=(\d+:\d+:\d+.\d+)`)

    for scanner.Scan() {
        line := scanner.Text()

        if matches := timeRegex.FindStringSubmatch(line); len(matches) > 1 {
            currentTime := parseFFmpegTime(matches[1])
            // 假设总时长已知
            progress := int(currentTime / task.Duration * 80) + 15 // 15-95%
            if progress > 95 {
                progress = 95
            }
            w.service.UpdateProgress(ctx, task.ID, progress, "Rendering...")
        }
    }
}
```

### 3.5 核心接口

```go
// service.go

type Service interface {
    // 任务管理
    CreateTask(ctx context.Context, userID uuid.UUID, req *CreateTaskRequest) (*RenderTask, error)
    GetTask(ctx context.Context, taskID uuid.UUID) (*RenderTask, error)
    CancelTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error
    ListTasks(ctx context.Context, userID uuid.UUID, filter *TaskFilter) (*TaskList, error)
    GetQueuePosition(ctx context.Context, taskID uuid.UUID) (int, error)

    // 进度
    UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status TaskStatus, errorMsg string) error
    UpdateProgress(ctx context.Context, taskID uuid.UUID, progress int, step string) error
    CompleteTask(ctx context.Context, taskID uuid.UUID, result *CompleteTaskResult) error

    // SSE
    SubscribeProgress(ctx context.Context, taskID uuid.UUID) (<-chan *ProgressEvent, error)

    // 产物
    GetDownloadURL(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (string, error)
    CleanExpiredOutputs(ctx context.Context) error

    // 预设
    GetPresets(ctx context.Context, userID uuid.UUID) ([]*RenderPreset, error)
    CreatePreset(ctx context.Context, userID uuid.UUID, req *CreatePresetRequest) (*RenderPreset, error)

    // 成本估算
    EstimateCost(ctx context.Context, req *EstimateCostRequest) (*CostEstimate, error)
}

type CreateTaskRequest struct {
    ProjectRepoID uuid.UUID     `json:"project_repo_id" binding:"required"`
    Format        OutputFormat  `json:"format"`
    Quality       OutputQuality `json:"quality"`
    FPS           int           `json:"fps"`
    PresetID      *uuid.UUID    `json:"preset_id"`
}

type CompleteTaskResult struct {
    OutputURL       string
    OutputSizeBytes int64
    ThumbnailURL    string
    Duration        float64
}

type ProgressEvent struct {
    TaskID   uuid.UUID  `json:"task_id"`
    Status   TaskStatus `json:"status"`
    Progress int        `json:"progress"`
    Step     string     `json:"step"`
    Error    string     `json:"error,omitempty"`
}

type CostEstimate struct {
    DurationSeconds float64 `json:"duration_seconds"`
    Quality         string  `json:"quality"`
    Multiplier      float64 `json:"multiplier"`
    EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}
```

### 3.6 SSE 进度推送

```go
// sse.go

type SSEHandler struct {
    service Service
    logger  *zap.Logger
}

func NewSSEHandler(service Service, logger *zap.Logger) *SSEHandler {
    return &SSEHandler{
        service: service,
        logger:  logger,
    }
}

// HandleProgress GET /api/v1/render/tasks/{id}/events
func (h *SSEHandler) HandleProgress(c *gin.Context) {
    taskID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
        return
    }

    // 设置 SSE 头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no") // Nginx 禁用缓冲

    // 订阅进度
    events, err := h.service.SubscribeProgress(c.Request.Context(), taskID)
    if err != nil {
        c.SSEvent("error", gin.H{"error": err.Error()})
        return
    }

    // 发送心跳和事件
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-c.Request.Context().Done():
            return

        case event, ok := <-events:
            if !ok {
                return
            }

            data, _ := json.Marshal(event)
            c.SSEvent("progress", string(data))
            c.Writer.Flush()

            // 任务完成或失败，关闭连接
            if event.Status == TaskStatusCompleted || event.Status == TaskStatusFailed {
                return
            }

        case <-ticker.C:
            // 心跳
            c.SSEvent("ping", time.Now().Unix())
            c.Writer.Flush()
        }
    }
}
```

### 3.7 计费规则

```go
// billing.go

type RenderBilling struct {
    logger *zap.Logger
}

// 计费规则
var QualityMultipliers = map[OutputQuality]float64{
    OutputQuality720p:  0.5,
    OutputQuality1080p: 1.0,
    OutputQuality4K:    2.5,
}

const (
    BasePricePerMinute = 0.05 // $0.05 per minute
    PriorityMultiplier = 1.5  // 优先队列加价 50%
)

func (b *RenderBilling) EstimateCost(duration float64, quality OutputQuality, priority bool) float64 {
    minutes := duration / 60
    multiplier := QualityMultipliers[quality]
    cost := minutes * BasePricePerMinute * multiplier

    if priority {
        cost *= PriorityMultiplier
    }

    return math.Round(cost*100) / 100 // 保留两位小数
}
```

### 3.8 API 接口

```yaml
# Render Task API

POST /api/v1/render/tasks:
  description: 创建渲染任务
  request:
    project_repo_id: uuid
    format: mp4 | webm | mov
    quality: 720p | 1080p | 4k
    fps: int
    preset_id: uuid (optional)
  response:
    RenderTask

GET /api/v1/render/tasks:
  description: 任务列表
  query:
    status: pending | processing | completed | failed
    page: int
  response:
    items: RenderTask[]
    total: int

GET /api/v1/render/tasks/{id}:
  description: 任务详情
  response:
    RenderTask with queue_position

DELETE /api/v1/render/tasks/{id}:
  description: 取消任务

GET /api/v1/render/tasks/{id}/events:
  description: SSE 进度推送
  headers:
    Accept: text/event-stream
  response:
    event: progress
    data:
      task_id: uuid
      status: string
      progress: int
      step: string

GET /api/v1/render/tasks/{id}/download:
  description: 获取下载链接
  response:
    url: string (presigned URL)
    expires_in: int

# Preset API

GET /api/v1/render/presets:
  description: 渲染预设列表
  response:
    items: RenderPreset[]

POST /api/v1/render/presets:
  description: 创建预设
  request:
    name: string
    format: string
    quality: string
    fps: int
    codec: string

# Cost Estimate

POST /api/v1/render/estimate:
  description: 估算成本
  request:
    project_repo_id: uuid
    quality: string
    priority: bool
  response:
    duration_seconds: float
    quality: string
    estimated_cost_usd: float
```

---

## 四、Publish Module

### 4.1 目录结构

```
internal/module/publish/
├── handler.go              # HTTP Handler
├── service.go              # 业务逻辑
├── repository.go           # 数据访问
├── model.go                # 数据模型
├── dto.go                  # 请求/响应 DTO
├── scheduler.go            # 定时发布调度
├── analytics.go            # 数据同步
├── platforms/              # 平台适配器
│   ├── interface.go        # 平台接口
│   ├── youtube.go          # YouTube
│   ├── bilibili.go         # Bilibili
│   ├── tiktok.go           # TikTok
│   └── factory.go          # 适配器工厂
└── errors.go               # 模块错误
```

### 4.2 数据模型

```go
// model.go

type Platform string

const (
    PlatformYouTube  Platform = "youtube"
    PlatformBilibili Platform = "bilibili"
    PlatformTikTok   Platform = "tiktok"
    PlatformDouyin   Platform = "douyin"
)

type PublishStatus string

const (
    PublishStatusPending    PublishStatus = "pending"
    PublishStatusScheduled  PublishStatus = "scheduled"
    PublishStatusUploading  PublishStatus = "uploading"
    PublishStatusProcessing PublishStatus = "processing"
    PublishStatusPublished  PublishStatus = "published"
    PublishStatusFailed     PublishStatus = "failed"
)

// PlatformAuth 平台授权
type PlatformAuth struct {
    ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID           uuid.UUID `gorm:"type:uuid;not null;index"`
    Platform         Platform  `gorm:"not null"`
    PlatformUserID   string    `gorm:"not null"`
    PlatformUsername string    `gorm:"not null"`
    AvatarURL        string
    AccessToken      string    `gorm:"not null"` // AES-256 加密
    RefreshToken     string                      // AES-256 加密
    ExpiresAt        time.Time
    Scopes           []string  `gorm:"type:text[]"`
    CreatedAt        time.Time `gorm:"autoCreateTime"`
    UpdatedAt        time.Time `gorm:"autoUpdateTime"`

    // Relations
    User *auth.User `gorm:"foreignKey:UserID"`
}

func (PlatformAuth) TableName() string { return "platform_auths" }

// PublishTask 发布任务
type PublishTask struct {
    ID            uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID        uuid.UUID     `gorm:"type:uuid;not null;index"`
    RenderTaskID  uuid.UUID     `gorm:"type:uuid;not null"`
    Platforms     []Platform    `gorm:"type:text[];not null"`
    Status        PublishStatus `gorm:"not null;default:pending"`
    ScheduledAt   *time.Time    // 定时发布时间
    CreatedAt     time.Time     `gorm:"autoCreateTime"`
    UpdatedAt     time.Time     `gorm:"autoUpdateTime"`

    // Relations
    Records []PublishRecord `gorm:"foreignKey:TaskID"`
}

func (PublishTask) TableName() string { return "publish_tasks" }

// PublishRecord 发布记录
type PublishRecord struct {
    ID              uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TaskID          uuid.UUID     `gorm:"type:uuid;not null;index"`
    Platform        Platform      `gorm:"not null"`
    PlatformVideoID string        // 平台视频 ID
    PlatformURL     string        // 平台视频链接
    Status          PublishStatus `gorm:"not null;default:pending"`
    ErrorMessage    string
    PublishedAt     *time.Time
    CreatedAt       time.Time     `gorm:"autoCreateTime"`
    UpdatedAt       time.Time     `gorm:"autoUpdateTime"`

    // Relations
    Config    *PublishConfig   `gorm:"foreignKey:RecordID"`
    Analytics *VideoAnalytics  `gorm:"foreignKey:RecordID"`
}

func (PublishRecord) TableName() string { return "publish_records" }

// PublishConfig 发布配置
type PublishConfig struct {
    RecordID    uuid.UUID `gorm:"type:uuid;primaryKey"`
    Title       string    `gorm:"not null"`
    Description string
    Tags        []string  `gorm:"type:text[]"`
    CoverURL    string    // 封面图
    Visibility  string    `gorm:"default:public"` // public, private, unlisted
    Category    string
    Playlist    string    // YouTube 播放列表
}

func (PublishConfig) TableName() string { return "publish_configs" }

// VideoAnalytics 视频数据
type VideoAnalytics struct {
    ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    RecordID         uuid.UUID `gorm:"type:uuid;not null;index"`
    Views            int64     `gorm:"default:0"`
    Likes            int       `gorm:"default:0"`
    Dislikes         int       `gorm:"default:0"`
    Comments         int       `gorm:"default:0"`
    Shares           int       `gorm:"default:0"`
    WatchTimeSeconds int64     `gorm:"default:0"` // 总观看时长
    AvgViewDuration  float64   `gorm:"default:0"` // 平均观看时长
    FetchedAt        time.Time `gorm:"autoUpdateTime"`
}

func (VideoAnalytics) TableName() string { return "video_analytics" }
```

### 4.3 平台适配器

```go
// platforms/interface.go

type PlatformAdapter interface {
    // Platform 返回平台类型
    Platform() Platform

    // GetAuthURL 获取 OAuth 授权 URL
    GetAuthURL(state string) string

    // HandleCallback 处理 OAuth 回调
    HandleCallback(ctx context.Context, code string) (*PlatformToken, error)

    // RefreshToken 刷新令牌
    RefreshToken(ctx context.Context, refreshToken string) (*PlatformToken, error)

    // GetUserInfo 获取用户信息
    GetUserInfo(ctx context.Context, accessToken string) (*PlatformUser, error)

    // Upload 上传视频
    Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error)

    // GetVideoStatus 获取视频处理状态
    GetVideoStatus(ctx context.Context, videoID string) (*VideoStatus, error)

    // GetAnalytics 获取视频数据
    GetAnalytics(ctx context.Context, videoID string) (*VideoAnalyticsData, error)

    // DeleteVideo 删除视频
    DeleteVideo(ctx context.Context, videoID string) error
}

type PlatformToken struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    Scopes       []string
}

type PlatformUser struct {
    ID        string
    Username  string
    AvatarURL string
}

type UploadRequest struct {
    AccessToken string
    VideoPath   string // 本地视频路径或 URL
    Config      *PublishConfig
    Progress    func(percent int) // 进度回调
}

type UploadResult struct {
    VideoID   string
    VideoURL  string
    Status    string
}

type VideoStatus struct {
    VideoID   string
    Status    string // processing, ready, failed
    Error     string
}

type VideoAnalyticsData struct {
    Views            int64
    Likes            int
    Comments         int
    Shares           int
    WatchTimeSeconds int64
}

// platforms/youtube.go

type YouTubeAdapter struct {
    clientID     string
    clientSecret string
    redirectURL  string
    apiKey       string
    client       *http.Client
    logger       *zap.Logger
}

func NewYouTubeAdapter(cfg *config.YouTubeConfig, logger *zap.Logger) *YouTubeAdapter {
    return &YouTubeAdapter{
        clientID:     cfg.ClientID,
        clientSecret: cfg.ClientSecret,
        redirectURL:  cfg.RedirectURL,
        apiKey:       cfg.APIKey,
        client:       &http.Client{Timeout: 120 * time.Second},
        logger:       logger,
    }
}

func (a *YouTubeAdapter) Platform() Platform {
    return PlatformYouTube
}

func (a *YouTubeAdapter) GetAuthURL(state string) string {
    params := url.Values{
        "client_id":     {a.clientID},
        "redirect_uri":  {a.redirectURL},
        "response_type": {"code"},
        "scope":         {"https://www.googleapis.com/auth/youtube.upload https://www.googleapis.com/auth/youtube.readonly"},
        "access_type":   {"offline"},
        "prompt":        {"consent"},
        "state":         {state},
    }
    return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func (a *YouTubeAdapter) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
    // 使用 YouTube Data API v3 的 resumable upload

    // 1. 初始化上传
    initURL := "https://www.googleapis.com/upload/youtube/v3/videos?uploadType=resumable&part=snippet,status"

    metadata := map[string]interface{}{
        "snippet": map[string]interface{}{
            "title":       req.Config.Title,
            "description": req.Config.Description,
            "tags":        req.Config.Tags,
            "categoryId":  req.Config.Category,
        },
        "status": map[string]interface{}{
            "privacyStatus": req.Config.Visibility,
        },
    }

    body, _ := json.Marshal(metadata)
    initReq, _ := http.NewRequestWithContext(ctx, "POST", initURL, bytes.NewReader(body))
    initReq.Header.Set("Authorization", "Bearer "+req.AccessToken)
    initReq.Header.Set("Content-Type", "application/json")
    initReq.Header.Set("X-Upload-Content-Type", "video/*")

    initResp, err := a.client.Do(initReq)
    if err != nil {
        return nil, err
    }
    defer initResp.Body.Close()

    if initResp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(initResp.Body)
        return nil, fmt.Errorf("init upload failed: %s", string(body))
    }

    uploadURL := initResp.Header.Get("Location")

    // 2. 上传视频内容
    videoFile, err := os.Open(req.VideoPath)
    if err != nil {
        return nil, err
    }
    defer videoFile.Close()

    stat, _ := videoFile.Stat()
    uploadReq, _ := http.NewRequestWithContext(ctx, "PUT", uploadURL, videoFile)
    uploadReq.Header.Set("Content-Type", "video/*")
    uploadReq.ContentLength = stat.Size()

    uploadResp, err := a.client.Do(uploadReq)
    if err != nil {
        return nil, err
    }
    defer uploadResp.Body.Close()

    if uploadResp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(uploadResp.Body)
        return nil, fmt.Errorf("upload failed: %s", string(body))
    }

    var result struct {
        ID string `json:"id"`
    }
    json.NewDecoder(uploadResp.Body).Decode(&result)

    return &UploadResult{
        VideoID:  result.ID,
        VideoURL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", result.ID),
        Status:   "processing",
    }, nil
}
```

### 4.4 核心接口

```go
// service.go

type Service interface {
    // 平台授权
    GetAuthURL(ctx context.Context, platform Platform, state string) (string, error)
    HandleCallback(ctx context.Context, platform Platform, code string, userID uuid.UUID) (*PlatformAuth, error)
    GetPlatformAuths(ctx context.Context, userID uuid.UUID) ([]*PlatformAuth, error)
    RevokePlatformAuth(ctx context.Context, userID uuid.UUID, platform Platform) error
    RefreshAuth(ctx context.Context, authID uuid.UUID) error

    // 发布任务
    CreateTask(ctx context.Context, userID uuid.UUID, req *CreatePublishTaskRequest) (*PublishTask, error)
    GetTask(ctx context.Context, taskID uuid.UUID) (*PublishTask, error)
    CancelTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error
    ListTasks(ctx context.Context, userID uuid.UUID, filter *TaskFilter) (*PublishTaskList, error)
    RetryRecord(ctx context.Context, userID uuid.UUID, recordID uuid.UUID) error

    // 发布历史
    GetRecord(ctx context.Context, recordID uuid.UUID) (*PublishRecord, error)
    ListRecords(ctx context.Context, userID uuid.UUID, filter *RecordFilter) (*RecordList, error)

    // 数据同步
    SyncAnalytics(ctx context.Context, recordID uuid.UUID) (*VideoAnalytics, error)
    GetAnalytics(ctx context.Context, recordID uuid.UUID) (*VideoAnalytics, error)
    GetAnalyticsSummary(ctx context.Context, userID uuid.UUID, period string) (*AnalyticsSummary, error)

    // 定时任务
    ProcessScheduledTasks(ctx context.Context) error
    RefreshAllTokens(ctx context.Context) error
    SyncAllAnalytics(ctx context.Context) error
}

type CreatePublishTaskRequest struct {
    RenderTaskID uuid.UUID           `json:"render_task_id" binding:"required"`
    ScheduledAt  *time.Time          `json:"scheduled_at"`
    Configs      []PlatformPublishConfig `json:"configs" binding:"required,min=1"`
}

type PlatformPublishConfig struct {
    Platform    Platform `json:"platform" binding:"required"`
    Title       string   `json:"title" binding:"required"`
    Description string   `json:"description"`
    Tags        []string `json:"tags"`
    CoverURL    string   `json:"cover_url"`
    Visibility  string   `json:"visibility"`
    Category    string   `json:"category"`
}

type AnalyticsSummary struct {
    Period      string `json:"period"`
    TotalViews  int64  `json:"total_views"`
    TotalLikes  int    `json:"total_likes"`
    TotalShares int    `json:"total_shares"`
    ByPlatform  map[Platform]*PlatformAnalytics `json:"by_platform"`
}

type PlatformAnalytics struct {
    VideoCount int   `json:"video_count"`
    Views      int64 `json:"views"`
    Likes      int   `json:"likes"`
    Comments   int   `json:"comments"`
    Shares     int   `json:"shares"`
}
```

### 4.5 API 接口

```yaml
# Platform Auth API

GET /api/v1/publish/platforms:
  description: 已授权平台列表
  response:
    items:
      - platform: string
        username: string
        avatar_url: string
        expires_at: timestamp

POST /api/v1/publish/platforms/{platform}/auth:
  description: 开始平台授权
  response:
    auth_url: string
    state: string

GET /api/v1/publish/platforms/{platform}/callback:
  description: OAuth 回调
  query:
    code: string
    state: string

DELETE /api/v1/publish/platforms/{platform}:
  description: 解除授权

# Publish Task API

POST /api/v1/publish/tasks:
  description: 创建发布任务
  request:
    render_task_id: uuid
    scheduled_at: timestamp (optional)
    configs:
      - platform: youtube | bilibili | tiktok
        title: string
        description: string
        tags: string[]
        visibility: public | private | unlisted
  response:
    PublishTask

GET /api/v1/publish/tasks:
  description: 任务列表
  query:
    status: pending | published | failed
    page: int

GET /api/v1/publish/tasks/{id}:
  description: 任务详情
  response:
    PublishTask with records

DELETE /api/v1/publish/tasks/{id}:
  description: 取消任务

POST /api/v1/publish/records/{id}/retry:
  description: 重试失败的发布

# Analytics API

GET /api/v1/publish/analytics/{record_id}:
  description: 单个视频数据
  response:
    VideoAnalytics

GET /api/v1/publish/analytics/summary:
  description: 数据汇总
  query:
    period: week | month | all
  response:
    AnalyticsSummary

POST /api/v1/publish/analytics/{record_id}/sync:
  description: 手动同步数据
  response:
    VideoAnalytics
```

---

## 五、数据库设计

### 5.1 迁移文件

```sql
-- migrations/000020_create_follows.up.sql

CREATE TABLE follows (
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id)
);

CREATE INDEX idx_follows_follower ON follows(follower_id);
CREATE INDEX idx_follows_following ON follows(following_id);

-- migrations/000021_create_likes.up.sql

CREATE TABLE likes (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, target_type, target_id)
);

CREATE INDEX idx_likes_target ON likes(target_type, target_id);

-- migrations/000022_create_comments.up.sql

CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    likes_count INT DEFAULT 0,
    replies_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_target ON comments(target_type, target_id);
CREATE INDEX idx_comments_user ON comments(user_id);
CREATE INDEX idx_comments_parent ON comments(parent_id);

-- migrations/000023_create_collections.up.sql

CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    visibility VARCHAR(50) DEFAULT 'private',
    items_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_collections_user ON collections(user_id);

CREATE TABLE collection_items (
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (collection_id, target_type, target_id)
);

-- migrations/000024_create_notifications.up.sql

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    content TEXT,
    read BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id) WHERE read = false;

-- migrations/000025_create_trending_cache.up.sql

CREATE TABLE trending_cache (
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    period VARCHAR(50) NOT NULL,
    score DECIMAL(10, 4) NOT NULL,
    rank INT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (target_type, target_id, period)
);

CREATE INDEX idx_trending_cache_rank ON trending_cache(target_type, period, rank);

-- migrations/000030_create_render_tasks.up.sql

CREATE TABLE render_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_repo_id UUID NOT NULL REFERENCES git_repos(id) ON DELETE CASCADE,
    project_commit VARCHAR(40) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority INT DEFAULT 0,
    output_format VARCHAR(50) NOT NULL DEFAULT 'mp4',
    output_quality VARCHAR(50) NOT NULL DEFAULT '1080p',
    output_fps INT DEFAULT 30,
    progress INT DEFAULT 0,
    current_step VARCHAR(255),
    output_url TEXT,
    output_size_bytes BIGINT,
    thumbnail_url TEXT,
    duration DECIMAL(10, 2),
    expires_at TIMESTAMPTZ,
    worker_id VARCHAR(255),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    estimated_cost DECIMAL(10, 4),
    actual_cost DECIMAL(10, 4),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_render_tasks_user ON render_tasks(user_id);
CREATE INDEX idx_render_tasks_status ON render_tasks(status);
CREATE INDEX idx_render_tasks_priority ON render_tasks(priority DESC, created_at ASC);

-- migrations/000040_create_platform_auths.up.sql

CREATE TABLE platform_auths (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    platform_username VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMPTZ,
    scopes TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, platform)
);

CREATE INDEX idx_platform_auths_user ON platform_auths(user_id);

-- migrations/000041_create_publish_tasks.up.sql

CREATE TABLE publish_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    render_task_id UUID NOT NULL REFERENCES render_tasks(id) ON DELETE CASCADE,
    platforms TEXT[] NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    scheduled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_publish_tasks_user ON publish_tasks(user_id);
CREATE INDEX idx_publish_tasks_scheduled ON publish_tasks(scheduled_at) WHERE scheduled_at IS NOT NULL;

CREATE TABLE publish_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES publish_tasks(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    platform_video_id VARCHAR(255),
    platform_url TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_publish_records_task ON publish_records(task_id);

CREATE TABLE publish_configs (
    record_id UUID PRIMARY KEY REFERENCES publish_records(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    tags TEXT[],
    cover_url TEXT,
    visibility VARCHAR(50) DEFAULT 'public',
    category VARCHAR(100),
    playlist VARCHAR(255)
);

CREATE TABLE video_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    record_id UUID NOT NULL REFERENCES publish_records(id) ON DELETE CASCADE,
    views BIGINT DEFAULT 0,
    likes INT DEFAULT 0,
    dislikes INT DEFAULT 0,
    comments INT DEFAULT 0,
    shares INT DEFAULT 0,
    watch_time_seconds BIGINT DEFAULT 0,
    avg_view_duration DECIMAL(10, 2) DEFAULT 0,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_video_analytics_record ON video_analytics(record_id);
```

---

## 六、定时任务

### 6.1 任务列表

| 任务 | 周期 | 说明 |
|------|------|------|
| 热榜计算 | 每小时 | 计算 daily/weekly/monthly 热榜 |
| Token 刷新 | 每 6 小时 | 刷新即将过期的平台授权 |
| 数据同步 | 每天 | 同步所有视频的播放数据 |
| 产物清理 | 每天 | 清理过期的渲染产物 |
| 通知清理 | 每周 | 清理 30 天前已读通知 |
| 定时发布 | 每分钟 | 检查并执行定时发布任务 |

### 6.2 实现

```go
// scheduler/scheduler.go

type Scheduler struct {
    community CommunityService
    render    RenderService
    publish   PublishService
    logger    *zap.Logger
}

func (s *Scheduler) Start(ctx context.Context) {
    // 热榜计算 - 每小时
    go s.runPeriodic(ctx, time.Hour, "trending", func() {
        s.community.RefreshTrending(context.Background())
    })

    // Token 刷新 - 每 6 小时
    go s.runPeriodic(ctx, 6*time.Hour, "token_refresh", func() {
        s.publish.RefreshAllTokens(context.Background())
    })

    // 数据同步 - 每天
    go s.runDaily(ctx, "03:00", "analytics_sync", func() {
        s.publish.SyncAllAnalytics(context.Background())
    })

    // 产物清理 - 每天
    go s.runDaily(ctx, "04:00", "output_cleanup", func() {
        s.render.CleanExpiredOutputs(context.Background())
    })

    // 定时发布 - 每分钟
    go s.runPeriodic(ctx, time.Minute, "scheduled_publish", func() {
        s.publish.ProcessScheduledTasks(context.Background())
    })
}

func (s *Scheduler) runPeriodic(ctx context.Context, interval time.Duration, name string, fn func()) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.logger.Info("running scheduled task", zap.String("name", name))
            fn()
        }
    }
}
```

---

## 七、验收标准

### 7.1 功能验收

| 模块 | 功能 | 验收条件 |
|------|------|----------|
| Community | 关注/取关 | 计数正确，通知发送 |
| Community | 点赞/评论 | 操作正常，计数更新 |
| Community | 热榜 | 定时刷新，排名合理 |
| Community | 推荐 | 个性化内容返回 |
| Render | 任务提交 | 入队成功，状态更新 |
| Render | 进度推送 | SSE 连接稳定 |
| Render | 产物下载 | 预签名 URL 有效 |
| Publish | 平台授权 | OAuth 流程完整 |
| Publish | 视频发布 | 上传成功，返回链接 |
| Publish | 数据同步 | 定时同步正常 |

### 7.2 性能指标

| 指标 | 目标值 |
|------|--------|
| 热榜计算 | < 30s |
| 推荐响应 | < 500ms |
| SSE 延迟 | < 1s |
| 视频上传 | 受平台 API 限制 |

### 7.3 测试覆盖

| 模块 | 单元测试 | 集成测试 |
|------|----------|----------|
| Community | > 70% | 热榜计算 |
| Render | > 60% | 任务流程 |
| Publish | > 60% | OAuth 流程 |
