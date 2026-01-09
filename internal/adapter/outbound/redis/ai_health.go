package redis

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

const (
	providerHealthKeyPrefix = "ai:provider:health:"
	accountHealthKeyPrefix  = "ai:account:health:"
)

// aiProviderHealthCacheAdapter implements outbound.AIProviderHealthCachePort.
type aiProviderHealthCacheAdapter struct {
	client *redis.Client
}

// NewAIProviderHealthCacheAdapter creates a new AI provider health cache adapter.
func NewAIProviderHealthCacheAdapter(client *redis.Client) outbound.AIProviderHealthCachePort {
	return &aiProviderHealthCacheAdapter{client: client}
}

func (a *aiProviderHealthCacheAdapter) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (bool, error) {
	key := providerHealthKeyPrefix + providerID.String()
	val, err := a.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Not in cache, assume healthy
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

func (a *aiProviderHealthCacheAdapter) SetProviderHealth(ctx context.Context, providerID uuid.UUID, healthy bool, ttl time.Duration) error {
	key := providerHealthKeyPrefix + providerID.String()
	val := "0"
	if healthy {
		val = "1"
	}
	return a.client.Set(ctx, key, val, ttl).Err()
}

func (a *aiProviderHealthCacheAdapter) GetAccountHealth(ctx context.Context, accountID uuid.UUID) (model.AIHealthStatus, error) {
	key := accountHealthKeyPrefix + accountID.String()
	val, err := a.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Not in cache, assume healthy
		return model.AIHealthStatusHealthy, nil
	}
	if err != nil {
		return model.AIHealthStatusUnhealthy, err
	}
	return model.AIHealthStatus(val), nil
}

func (a *aiProviderHealthCacheAdapter) SetAccountHealth(ctx context.Context, accountID uuid.UUID, status model.AIHealthStatus, ttl time.Duration) error {
	key := accountHealthKeyPrefix + accountID.String()
	return a.client.Set(ctx, key, string(status), ttl).Err()
}

func (a *aiProviderHealthCacheAdapter) InvalidateProviderHealth(ctx context.Context, providerID uuid.UUID) error {
	key := providerHealthKeyPrefix + providerID.String()
	return a.client.Del(ctx, key).Err()
}

func (a *aiProviderHealthCacheAdapter) InvalidateAccountHealth(ctx context.Context, accountID uuid.UUID) error {
	key := accountHealthKeyPrefix + accountID.String()
	return a.client.Del(ctx, key).Err()
}

// Compile-time check
var _ outbound.AIProviderHealthCachePort = (*aiProviderHealthCacheAdapter)(nil)
