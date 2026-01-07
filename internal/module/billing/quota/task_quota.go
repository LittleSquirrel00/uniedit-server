package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/module/billing"
	"go.uber.org/zap"
)

// Task-specific quota errors
var (
	ErrChatQuotaExceeded      = fmt.Errorf("monthly chat token quota exceeded")
	ErrImageQuotaExceeded     = fmt.Errorf("monthly image credit quota exceeded")
	ErrVideoQuotaExceeded     = fmt.Errorf("monthly video minutes quota exceeded")
	ErrEmbeddingQuotaExceeded = fmt.Errorf("monthly embedding token quota exceeded")
)

// TaskQuotaChecker checks task-specific quotas.
type TaskQuotaChecker struct {
	redis          *redis.Client
	billingService billing.ServiceInterface
	logger         *zap.Logger
}

// NewTaskQuotaChecker creates a new task quota checker.
func NewTaskQuotaChecker(redis *redis.Client, billingService billing.ServiceInterface, logger *zap.Logger) *TaskQuotaChecker {
	return &TaskQuotaChecker{
		redis:          redis,
		billingService: billingService,
		logger:         logger,
	}
}

// CheckChatQuota checks if user has chat token quota available.
func (c *TaskQuotaChecker) CheckChatQuota(ctx context.Context, userID uuid.UUID, estimatedTokens int64) error {
	sub, err := c.billingService.GetSubscription(ctx, userID)
	if err != nil {
		c.logger.Warn("failed to get subscription, allowing request", zap.Error(err))
		return nil
	}

	plan := sub.Plan()
	if plan == nil {
		return nil
	}

	// Get effective limit (backward compatible)
	limit := plan.GetEffectiveChatTokenLimit()
	if limit == -1 {
		return nil // Unlimited
	}

	// Get current usage
	used, err := c.GetChatTokensUsed(ctx, userID, sub.CurrentPeriodStart())
	if err != nil {
		c.logger.Warn("redis error, allowing request", zap.Error(err))
		return nil
	}

	if used+estimatedTokens > limit {
		return ErrChatQuotaExceeded
	}

	return nil
}

// CheckImageQuota checks if user has image credit quota available.
func (c *TaskQuotaChecker) CheckImageQuota(ctx context.Context, userID uuid.UUID) error {
	sub, err := c.billingService.GetSubscription(ctx, userID)
	if err != nil {
		c.logger.Warn("failed to get subscription, allowing request", zap.Error(err))
		return nil
	}

	plan := sub.Plan()
	if plan == nil || plan.IsUnlimitedImageCredits() {
		return nil
	}

	used, err := c.GetImageCreditsUsed(ctx, userID, sub.CurrentPeriodStart())
	if err != nil {
		c.logger.Warn("redis error, allowing request", zap.Error(err))
		return nil
	}

	if used >= int64(plan.MonthlyImageCredits()) {
		return ErrImageQuotaExceeded
	}

	return nil
}

// CheckVideoQuota checks if user has video minutes quota available.
func (c *TaskQuotaChecker) CheckVideoQuota(ctx context.Context, userID uuid.UUID, estimatedMinutes int) error {
	sub, err := c.billingService.GetSubscription(ctx, userID)
	if err != nil {
		c.logger.Warn("failed to get subscription, allowing request", zap.Error(err))
		return nil
	}

	plan := sub.Plan()
	if plan == nil || plan.IsUnlimitedVideoMinutes() {
		return nil
	}

	used, err := c.GetVideoMinutesUsed(ctx, userID, sub.CurrentPeriodStart())
	if err != nil {
		c.logger.Warn("redis error, allowing request", zap.Error(err))
		return nil
	}

	if int(used)+estimatedMinutes > plan.MonthlyVideoMinutes() {
		return ErrVideoQuotaExceeded
	}

	return nil
}

// CheckEmbeddingQuota checks if user has embedding token quota available.
func (c *TaskQuotaChecker) CheckEmbeddingQuota(ctx context.Context, userID uuid.UUID, estimatedTokens int64) error {
	sub, err := c.billingService.GetSubscription(ctx, userID)
	if err != nil {
		c.logger.Warn("failed to get subscription, allowing request", zap.Error(err))
		return nil
	}

	plan := sub.Plan()
	if plan == nil || plan.IsUnlimitedEmbeddingTokens() {
		return nil
	}

	used, err := c.GetEmbeddingTokensUsed(ctx, userID, sub.CurrentPeriodStart())
	if err != nil {
		c.logger.Warn("redis error, allowing request", zap.Error(err))
		return nil
	}

	if used+estimatedTokens > plan.MonthlyEmbeddingTokens() {
		return ErrEmbeddingQuotaExceeded
	}

	return nil
}

// GetChatTokensUsed returns chat tokens used this period.
func (c *TaskQuotaChecker) GetChatTokensUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error) {
	key := chatTokenKey(userID, periodStart)
	return c.getInt64(ctx, key)
}

// GetImageCreditsUsed returns image credits used this period.
func (c *TaskQuotaChecker) GetImageCreditsUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error) {
	key := imageCreditsKey(userID, periodStart)
	return c.getInt64(ctx, key)
}

// GetVideoMinutesUsed returns video minutes used this period.
func (c *TaskQuotaChecker) GetVideoMinutesUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error) {
	key := videoMinutesKey(userID, periodStart)
	return c.getInt64(ctx, key)
}

// GetEmbeddingTokensUsed returns embedding tokens used this period.
func (c *TaskQuotaChecker) GetEmbeddingTokensUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error) {
	key := embeddingTokenKey(userID, periodStart)
	return c.getInt64(ctx, key)
}

// IncrementChatTokens increments chat token counter.
func (c *TaskQuotaChecker) IncrementChatTokens(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, tokens int64) error {
	key := chatTokenKey(userID, periodStart)
	return c.increment(ctx, key, tokens, periodEnd)
}

// IncrementImageCredits increments image credit counter.
func (c *TaskQuotaChecker) IncrementImageCredits(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, count int64) error {
	key := imageCreditsKey(userID, periodStart)
	return c.increment(ctx, key, count, periodEnd)
}

// IncrementVideoMinutes increments video minutes counter.
func (c *TaskQuotaChecker) IncrementVideoMinutes(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, minutes int64) error {
	key := videoMinutesKey(userID, periodStart)
	return c.increment(ctx, key, minutes, periodEnd)
}

// IncrementEmbeddingTokens increments embedding token counter.
func (c *TaskQuotaChecker) IncrementEmbeddingTokens(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, tokens int64) error {
	key := embeddingTokenKey(userID, periodStart)
	return c.increment(ctx, key, tokens, periodEnd)
}

// helper methods
func (c *TaskQuotaChecker) getInt64(ctx context.Context, key string) (int64, error) {
	val, err := c.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (c *TaskQuotaChecker) increment(ctx context.Context, key string, amount int64, periodEnd time.Time) error {
	_, err := c.redis.IncrBy(ctx, key, amount).Result()
	if err != nil {
		return err
	}

	ttl := time.Until(periodEnd)
	if ttl > 0 {
		c.redis.Expire(ctx, key, ttl)
	}
	return nil
}

// Key helpers for task-specific quotas
func chatTokenKey(userID uuid.UUID, periodStart time.Time) string {
	return fmt.Sprintf("quota:chat:%s:%s", userID.String(), periodStart.Format("2006-01"))
}

func imageCreditsKey(userID uuid.UUID, periodStart time.Time) string {
	return fmt.Sprintf("quota:image:%s:%s", userID.String(), periodStart.Format("2006-01"))
}

func videoMinutesKey(userID uuid.UUID, periodStart time.Time) string {
	return fmt.Sprintf("quota:video:%s:%s", userID.String(), periodStart.Format("2006-01"))
}

func embeddingTokenKey(userID uuid.UUID, periodStart time.Time) string {
	return fmt.Sprintf("quota:embedding:%s:%s", userID.String(), periodStart.Format("2006-01"))
}
