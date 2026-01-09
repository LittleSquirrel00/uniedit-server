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
	mediaHealthKeyPrefix = "media:health:"
	mediaHealthTTL       = 5 * time.Minute
)

// MediaHealthCacheAdapter implements MediaProviderHealthCachePort.
type MediaHealthCacheAdapter struct {
	client redis.UniversalClient
}

// NewMediaHealthCacheAdapter creates a new media health cache adapter.
func NewMediaHealthCacheAdapter(client redis.UniversalClient) *MediaHealthCacheAdapter {
	return &MediaHealthCacheAdapter{client: client}
}

func (a *MediaHealthCacheAdapter) GetHealth(ctx context.Context, providerID uuid.UUID) (bool, error) {
	key := mediaHealthKeyPrefix + providerID.String()
	val, err := a.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key not found, assume healthy
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("get health: %w", err)
	}
	return val == "1", nil
}

func (a *MediaHealthCacheAdapter) SetHealth(ctx context.Context, providerID uuid.UUID, healthy bool) error {
	key := mediaHealthKeyPrefix + providerID.String()
	val := "0"
	if healthy {
		val = "1"
	}
	if err := a.client.Set(ctx, key, val, mediaHealthTTL).Err(); err != nil {
		return fmt.Errorf("set health: %w", err)
	}
	return nil
}

// Compile-time interface check
var _ outbound.MediaProviderHealthCachePort = (*MediaHealthCacheAdapter)(nil)
