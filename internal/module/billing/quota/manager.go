package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Manager manages quota tracking and enforcement using Redis.
type Manager struct {
	redis  *redis.Client
	logger *zap.Logger
}

// NewManager creates a new quota manager.
func NewManager(redis *redis.Client, logger *zap.Logger) *Manager {
	return &Manager{
		redis:  redis,
		logger: logger,
	}
}

// GetTokensUsed returns the number of tokens used this period.
func (m *Manager) GetTokensUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error) {
	key := tokenKey(userID, periodStart)
	val, err := m.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		m.logger.Error("failed to get tokens used", zap.Error(err), zap.String("key", key))
		return 0, err
	}
	return val, nil
}

// GetRequestsToday returns the number of requests made today.
func (m *Manager) GetRequestsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	key := requestKey(userID, time.Now().UTC())
	val, err := m.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		m.logger.Error("failed to get requests today", zap.Error(err), zap.String("key", key))
		return 0, err
	}
	return val, nil
}

// IncrementTokens increments the token counter for the period.
func (m *Manager) IncrementTokens(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, tokens int64) (int64, error) {
	key := tokenKey(userID, periodStart)

	// Use INCRBY and set expiration
	val, err := m.redis.IncrBy(ctx, key, tokens).Result()
	if err != nil {
		m.logger.Error("failed to increment tokens", zap.Error(err), zap.String("key", key))
		return 0, err
	}

	// Set expiration if this is a new key
	ttl := time.Until(periodEnd)
	if ttl > 0 {
		m.redis.Expire(ctx, key, ttl)
	}

	return val, nil
}

// IncrementRequests increments the daily request counter.
func (m *Manager) IncrementRequests(ctx context.Context, userID uuid.UUID) (int, error) {
	now := time.Now().UTC()
	key := requestKey(userID, now)

	// Use INCR and set expiration
	val, err := m.redis.Incr(ctx, key).Result()
	if err != nil {
		m.logger.Error("failed to increment requests", zap.Error(err), zap.String("key", key))
		return 0, err
	}

	// Set expiration to end of day UTC
	endOfDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	ttl := time.Until(endOfDay)
	if ttl > 0 {
		m.redis.Expire(ctx, key, ttl)
	}

	return int(val), nil
}

// ResetTokens resets the token counter for a user.
func (m *Manager) ResetTokens(ctx context.Context, userID uuid.UUID, periodStart time.Time) error {
	key := tokenKey(userID, periodStart)
	return m.redis.Del(ctx, key).Err()
}

// CheckQuota checks if the user has quota available.
// Returns nil if quota is available, error otherwise.
func (m *Manager) CheckQuota(ctx context.Context, userID uuid.UUID, periodStart time.Time, tokenLimit int64, requestLimit int) error {
	// Check token limit (skip if unlimited)
	if tokenLimit != -1 {
		tokensUsed, err := m.GetTokensUsed(ctx, userID, periodStart)
		if err != nil {
			// Graceful degradation: allow request on Redis error
			m.logger.Warn("redis error during quota check, allowing request", zap.Error(err))
			return nil
		}
		if tokensUsed >= tokenLimit {
			return ErrTokenQuotaExceeded
		}
	}

	// Check request limit (skip if unlimited)
	if requestLimit != -1 {
		requestsToday, err := m.GetRequestsToday(ctx, userID)
		if err != nil {
			// Graceful degradation
			m.logger.Warn("redis error during quota check, allowing request", zap.Error(err))
			return nil
		}
		if requestsToday >= requestLimit {
			return ErrRequestQuotaExceeded
		}
	}

	return nil
}

// Errors
var (
	ErrTokenQuotaExceeded   = fmt.Errorf("monthly token quota exceeded")
	ErrRequestQuotaExceeded = fmt.Errorf("daily request quota exceeded")
)

// Key helpers
func tokenKey(userID uuid.UUID, periodStart time.Time) string {
	return fmt.Sprintf("quota:tokens:%s:%s", userID.String(), periodStart.Format("2006-01"))
}

func requestKey(userID uuid.UUID, date time.Time) string {
	return fmt.Sprintf("quota:requests:%s:%s", userID.String(), date.Format("2006-01-02"))
}
