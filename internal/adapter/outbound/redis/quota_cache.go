package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/port/outbound"
)

const (
	quotaTokensKeyPrefix   = "quota:tokens:"
	quotaRequestsKeyPrefix = "quota:requests:"
	quotaMediaUnitsKeyPrefix = "quota:media:"
)

// quotaCache implements outbound.QuotaCachePort.
type quotaCache struct {
	client *redis.Client
}

// NewQuotaCache creates a new quota cache adapter.
func NewQuotaCache(client *redis.Client) outbound.QuotaCachePort {
	return &quotaCache{client: client}
}

func (c *quotaCache) tokenKey(userID uuid.UUID, periodStart time.Time) string {
	return fmt.Sprintf("%s%s:%s", quotaTokensKeyPrefix, userID.String(), periodStart.Format("2006-01"))
}

func (c *quotaCache) requestKey(userID uuid.UUID) string {
	today := time.Now().UTC().Format("2006-01-02")
	return fmt.Sprintf("%s%s:%s", quotaRequestsKeyPrefix, userID.String(), today)
}

func (c *quotaCache) mediaUnitsKey(userID uuid.UUID, periodStart time.Time, taskType string) string {
	return fmt.Sprintf("%s%s:%s:%s", quotaMediaUnitsKeyPrefix, taskType, userID.String(), periodStart.Format("2006-01"))
}

func (c *quotaCache) GetTokensUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error) {
	key := c.tokenKey(userID, periodStart)
	val, err := c.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, outbound.ErrCacheMiss
		}
		return 0, err
	}
	return val, nil
}

func (c *quotaCache) IncrementTokens(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, tokens int64) (int64, error) {
	key := c.tokenKey(userID, periodStart)

	// Increment the counter
	newVal, err := c.client.IncrBy(ctx, key, tokens).Result()
	if err != nil {
		return 0, err
	}

	// Set expiry to end of period plus a buffer
	ttl := time.Until(periodEnd) + 24*time.Hour
	if ttl > 0 {
		c.client.Expire(ctx, key, ttl)
	}

	return newVal, nil
}

func (c *quotaCache) GetRequestsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	key := c.requestKey(userID)
	val, err := c.client.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, outbound.ErrCacheMiss
		}
		return 0, err
	}
	return val, nil
}

func (c *quotaCache) IncrementRequests(ctx context.Context, userID uuid.UUID) (int, error) {
	key := c.requestKey(userID)

	// Increment the counter
	newVal, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// Set expiry to end of day (midnight UTC)
	now := time.Now().UTC()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	ttl := time.Until(endOfDay)
	if ttl > 0 {
		c.client.Expire(ctx, key, ttl)
	}

	return int(newVal), nil
}

func (c *quotaCache) ResetTokens(ctx context.Context, userID uuid.UUID, periodStart time.Time) error {
	key := c.tokenKey(userID, periodStart)
	return c.client.Del(ctx, key).Err()
}

func (c *quotaCache) GetMediaUnitsUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time, taskType string) (int64, error) {
	key := c.mediaUnitsKey(userID, periodStart, taskType)
	val, err := c.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, outbound.ErrCacheMiss
		}
		return 0, err
	}
	return val, nil
}

func (c *quotaCache) IncrementMediaUnits(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, taskType string, units int64) (int64, error) {
	key := c.mediaUnitsKey(userID, periodStart, taskType)

	newVal, err := c.client.IncrBy(ctx, key, units).Result()
	if err != nil {
		return 0, err
	}

	ttl := time.Until(periodEnd) + 24*time.Hour
	if ttl > 0 {
		c.client.Expire(ctx, key, ttl)
	}

	return newVal, nil
}

func (c *quotaCache) ResetMediaUnits(ctx context.Context, userID uuid.UUID, periodStart time.Time, taskType string) error {
	key := c.mediaUnitsKey(userID, periodStart, taskType)
	return c.client.Del(ctx, key).Err()
}

// Compile-time check
var _ outbound.QuotaCachePort = (*quotaCache)(nil)
