package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/module/billing"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Storage quota errors
var (
	ErrGitStorageQuotaExceeded = fmt.Errorf("git storage quota exceeded")
	ErrLFSStorageQuotaExceeded = fmt.Errorf("lfs storage quota exceeded")
)

// StorageUsage represents user's storage usage.
type StorageUsage struct {
	GitBytes int64 `json:"git_bytes"`
	LFSBytes int64 `json:"lfs_bytes"`
}

// StorageQuotaChecker checks storage quotas.
type StorageQuotaChecker struct {
	db             *gorm.DB
	redis          *redis.Client
	billingService billing.ServiceInterface
	logger         *zap.Logger
	cacheTTL       time.Duration
}

// NewStorageQuotaChecker creates a new storage quota checker.
func NewStorageQuotaChecker(db *gorm.DB, redis *redis.Client, billingService billing.ServiceInterface, logger *zap.Logger) *StorageQuotaChecker {
	return &StorageQuotaChecker{
		db:             db,
		redis:          redis,
		billingService: billingService,
		logger:         logger,
		cacheTTL:       5 * time.Minute,
	}
}

// CheckGitStorageQuota checks if user has git storage quota available.
func (c *StorageQuotaChecker) CheckGitStorageQuota(ctx context.Context, userID uuid.UUID, additionalBytes int64) error {
	sub, err := c.billingService.GetSubscription(ctx, userID)
	if err != nil {
		c.logger.Warn("failed to get subscription, allowing request", zap.Error(err))
		return nil
	}

	plan := sub.Plan()
	if plan == nil || plan.IsUnlimitedGitStorage() {
		return nil
	}

	usage, err := c.GetUserStorageUsage(ctx, userID)
	if err != nil {
		c.logger.Warn("failed to get storage usage, allowing request", zap.Error(err))
		return nil
	}

	limitBytes := plan.GitStorageMB() * 1024 * 1024
	if usage.GitBytes+additionalBytes > limitBytes {
		return ErrGitStorageQuotaExceeded
	}

	return nil
}

// CheckLFSStorageQuota checks if user has LFS storage quota available.
func (c *StorageQuotaChecker) CheckLFSStorageQuota(ctx context.Context, userID uuid.UUID, additionalBytes int64) error {
	sub, err := c.billingService.GetSubscription(ctx, userID)
	if err != nil {
		c.logger.Warn("failed to get subscription, allowing request", zap.Error(err))
		return nil
	}

	plan := sub.Plan()
	if plan == nil || plan.IsUnlimitedLFSStorage() {
		return nil
	}

	usage, err := c.GetUserStorageUsage(ctx, userID)
	if err != nil {
		c.logger.Warn("failed to get storage usage, allowing request", zap.Error(err))
		return nil
	}

	limitBytes := plan.LFSStorageMB() * 1024 * 1024
	if usage.LFSBytes+additionalBytes > limitBytes {
		return ErrLFSStorageQuotaExceeded
	}

	return nil
}

// GetUserStorageUsage returns user's storage usage (with caching).
func (c *StorageQuotaChecker) GetUserStorageUsage(ctx context.Context, userID uuid.UUID) (*StorageUsage, error) {
	// Try cache first
	if c.redis != nil {
		usage, err := c.getStorageFromCache(ctx, userID)
		if err == nil && usage != nil {
			return usage, nil
		}
	}

	// Calculate from database
	usage, err := c.calculateStorageUsage(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if c.redis != nil {
		c.setStorageCache(ctx, userID, usage)
	}

	return usage, nil
}

// InvalidateStorageCache invalidates storage cache for a user.
func (c *StorageQuotaChecker) InvalidateStorageCache(ctx context.Context, userID uuid.UUID) {
	if c.redis == nil {
		return
	}

	key := storageKey(userID)
	if err := c.redis.Del(ctx, key).Err(); err != nil {
		c.logger.Warn("failed to invalidate storage cache", zap.Error(err))
	}
}

// calculateStorageUsage calculates storage usage from database.
func (c *StorageQuotaChecker) calculateStorageUsage(ctx context.Context, userID uuid.UUID) (*StorageUsage, error) {
	var result struct {
		GitTotal int64
		LFSTotal int64
	}

	err := c.db.WithContext(ctx).
		Table("git_repos").
		Select("COALESCE(SUM(size_bytes), 0) as git_total, COALESCE(SUM(lfs_size_bytes), 0) as lfs_total").
		Where("owner_id = ?", userID).
		Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("calculate storage usage: %w", err)
	}

	return &StorageUsage{
		GitBytes: result.GitTotal,
		LFSBytes: result.LFSTotal,
	}, nil
}

// getStorageFromCache retrieves storage usage from cache.
func (c *StorageQuotaChecker) getStorageFromCache(ctx context.Context, userID uuid.UUID) (*StorageUsage, error) {
	key := storageKey(userID)

	gitBytes, err := c.redis.HGet(ctx, key, "git").Int64()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	lfsBytes, err := c.redis.HGet(ctx, key, "lfs").Int64()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	// Check if key exists
	exists, _ := c.redis.Exists(ctx, key).Result()
	if exists == 0 {
		return nil, nil
	}

	return &StorageUsage{
		GitBytes: gitBytes,
		LFSBytes: lfsBytes,
	}, nil
}

// setStorageCache stores storage usage in cache.
func (c *StorageQuotaChecker) setStorageCache(ctx context.Context, userID uuid.UUID, usage *StorageUsage) {
	key := storageKey(userID)

	pipe := c.redis.Pipeline()
	pipe.HSet(ctx, key, "git", usage.GitBytes)
	pipe.HSet(ctx, key, "lfs", usage.LFSBytes)
	pipe.Expire(ctx, key, c.cacheTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		c.logger.Warn("failed to cache storage usage", zap.Error(err))
	}
}

// storageKey returns the Redis key for storage cache.
func storageKey(userID uuid.UUID) string {
	return fmt.Sprintf("storage:user:%s", userID.String())
}
