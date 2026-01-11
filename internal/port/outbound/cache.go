package outbound

import (
	"context"
	"time"
)

// CachePort defines generic cache operations.
type CachePort interface {
	// Get retrieves a value from cache.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in cache with TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from cache.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in cache.
	Exists(ctx context.Context, key string) (bool, error)
}
