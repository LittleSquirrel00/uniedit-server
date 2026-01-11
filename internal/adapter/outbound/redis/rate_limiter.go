package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/port/outbound"
)

const rateLimitKeyPrefix = "ratelimit:"

// rateLimiter implements outbound.RateLimiterPort.
type rateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a new rate limiter adapter.
func NewRateLimiter(client *redis.Client) outbound.RateLimiterPort {
	return &rateLimiter{client: client}
}

func (r *rateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	return r.AllowN(ctx, key, 1, limit, window)
}

func (r *rateLimiter) AllowN(ctx context.Context, key string, n int, limit int, window time.Duration) (bool, error) {
	fullKey := rateLimitKeyPrefix + key

	// Use sliding window counter algorithm
	pipe := r.client.Pipeline()
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", windowStart))

	// Count current entries
	countCmd := pipe.ZCard(ctx, fullKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	currentCount := countCmd.Val()

	// Check if we can add more
	if currentCount+int64(n) > int64(limit) {
		return false, nil
	}

	// Add new entries
	members := make([]redis.Z, n)
	for i := 0; i < n; i++ {
		members[i] = redis.Z{
			Score:  float64(now + int64(i)),
			Member: fmt.Sprintf("%d-%d", now, i),
		}
	}

	pipe2 := r.client.Pipeline()
	pipe2.ZAdd(ctx, fullKey, members...)
	pipe2.Expire(ctx, fullKey, window)
	_, err = pipe2.Exec(ctx)

	return err == nil, err
}

func (r *rateLimiter) GetRemaining(ctx context.Context, key string, limit int, window time.Duration) (int, error) {
	fullKey := rateLimitKeyPrefix + key
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	// Remove old entries and count
	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, fullKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	remaining := limit - int(countCmd.Val())
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Compile-time check
var _ outbound.RateLimiterPort = (*rateLimiter)(nil)
