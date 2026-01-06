package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RateLimiter provides rate limiting functionality for API keys.
type RateLimiter struct {
	redis redis.UniversalClient
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(redis redis.UniversalClient) *RateLimiter {
	return &RateLimiter{redis: redis}
}

// RateLimitResult contains the result of a rate limit check.
type RateLimitResult struct {
	Allowed   bool  `json:"allowed"`
	Remaining int64 `json:"remaining"`
	ResetAt   int64 `json:"reset_at"`
	Limit     int64 `json:"limit"`
}

// CheckRPM checks if the request is allowed under the RPM (requests per minute) limit.
func (r *RateLimiter) CheckRPM(ctx context.Context, keyID uuid.UUID, limit int) (*RateLimitResult, error) {
	return r.checkLimit(ctx, fmt.Sprintf("ratelimit:rpm:%s", keyID.String()), int64(limit), time.Minute)
}

// CheckTPM checks if the request is allowed under the TPM (tokens per minute) limit.
func (r *RateLimiter) CheckTPM(ctx context.Context, keyID uuid.UUID, limit int, tokenCount int) (*RateLimitResult, error) {
	return r.checkLimitWithIncrement(ctx, fmt.Sprintf("ratelimit:tpm:%s", keyID.String()), int64(limit), time.Minute, int64(tokenCount))
}

// IncrementTPM increments the TPM counter after a request completes.
func (r *RateLimiter) IncrementTPM(ctx context.Context, keyID uuid.UUID, tokenCount int) error {
	key := fmt.Sprintf("ratelimit:tpm:%s", keyID.String())
	now := time.Now()
	windowEnd := now.Truncate(time.Minute).Add(time.Minute).Unix()

	pipe := r.redis.Pipeline()

	// Add tokens to sorted set with current timestamp
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d:%d", now.UnixNano(), tokenCount),
	})

	// Set expiry
	pipe.ExpireAt(ctx, key, time.Unix(windowEnd+60, 0))

	_, err := pipe.Exec(ctx)
	return err
}

// checkLimit performs a sliding window rate limit check.
func (r *RateLimiter) checkLimit(ctx context.Context, key string, limit int64, window time.Duration) (*RateLimitResult, error) {
	return r.checkLimitWithIncrement(ctx, key, limit, window, 1)
}

// checkLimitWithIncrement performs a sliding window rate limit check with a custom increment.
func (r *RateLimiter) checkLimitWithIncrement(ctx context.Context, key string, limit int64, window time.Duration, increment int64) (*RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-window).UnixNano()
	windowEnd := now.UnixNano()

	// Use Lua script for atomic operations
	script := redis.NewScript(`
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local window_end = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local increment = tonumber(ARGV[4])
		local expiry = tonumber(ARGV[5])

		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current requests/tokens in window
		local current = 0
		local entries = redis.call('ZRANGE', key, 0, -1)
		for i, entry in ipairs(entries) do
			local parts = {}
			for part in string.gmatch(entry, "[^:]+") do
				table.insert(parts, part)
			end
			if #parts == 2 then
				current = current + tonumber(parts[2])
			else
				current = current + 1
			end
		end

		-- Check if limit would be exceeded
		if current + increment > limit then
			return {0, limit - current, expiry}
		end

		-- Add new entry
		local member = window_end .. ':' .. increment
		redis.call('ZADD', key, window_end, member)
		redis.call('PEXPIRE', key, expiry)

		return {1, limit - current - increment, expiry}
	`)

	result, err := script.Run(ctx, r.redis, []string{key},
		windowStart,
		windowEnd,
		limit,
		increment,
		int64(window.Milliseconds())+60000, // Add buffer for cleanup
	).Slice()
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	allowed, _ := strconv.ParseInt(fmt.Sprint(result[0]), 10, 64)
	remaining, _ := strconv.ParseInt(fmt.Sprint(result[1]), 10, 64)
	resetAt := now.Add(window).Unix()

	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   allowed == 1,
		Remaining: remaining,
		ResetAt:   resetAt,
		Limit:     limit,
	}, nil
}

// GetRPMUsage returns the current RPM usage for an API key.
func (r *RateLimiter) GetRPMUsage(ctx context.Context, keyID uuid.UUID) (int64, error) {
	return r.getUsage(ctx, fmt.Sprintf("ratelimit:rpm:%s", keyID.String()))
}

// GetTPMUsage returns the current TPM usage for an API key.
func (r *RateLimiter) GetTPMUsage(ctx context.Context, keyID uuid.UUID) (int64, error) {
	return r.getUsage(ctx, fmt.Sprintf("ratelimit:tpm:%s", keyID.String()))
}

// getUsage returns the current usage count in the window.
func (r *RateLimiter) getUsage(ctx context.Context, key string) (int64, error) {
	now := time.Now()
	windowStart := now.Add(-time.Minute).UnixNano()

	// Remove old entries first
	r.redis.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprint(windowStart))

	// Count current usage
	entries, err := r.redis.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return 0, err
	}

	var total int64
	for _, entry := range entries {
		// Parse entry format: "timestamp:count"
		var count int64 = 1
		if _, err := fmt.Sscanf(entry, "%*d:%d", &count); err == nil && count > 0 {
			total += count
		} else {
			total++
		}
	}

	return total, nil
}

// Reset clears the rate limit counters for an API key.
func (r *RateLimiter) Reset(ctx context.Context, keyID uuid.UUID) error {
	pipe := r.redis.Pipeline()
	pipe.Del(ctx, fmt.Sprintf("ratelimit:rpm:%s", keyID.String()))
	pipe.Del(ctx, fmt.Sprintf("ratelimit:tpm:%s", keyID.String()))
	_, err := pipe.Exec(ctx)
	return err
}
