package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/shared/config"
)

// NewRedisClient creates a new Redis client.
func NewRedisClient(cfg *config.RedisConfig) (redis.UniversalClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Verify connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// Close closes the Redis client.
func Close(client redis.UniversalClient) error {
	return client.Close()
}
