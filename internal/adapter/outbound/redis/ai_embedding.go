package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/port/outbound"
)

const (
	embeddingKeyPrefix = "ai:embedding:"
)

// aiEmbeddingCacheAdapter implements outbound.AIEmbeddingCachePort.
type aiEmbeddingCacheAdapter struct {
	client *redis.Client
}

// NewAIEmbeddingCacheAdapter creates a new AI embedding cache adapter.
func NewAIEmbeddingCacheAdapter(client *redis.Client) outbound.AIEmbeddingCachePort {
	return &aiEmbeddingCacheAdapter{client: client}
}

func (a *aiEmbeddingCacheAdapter) Get(ctx context.Context, key string) ([]float64, error) {
	val, err := a.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var embedding []float64
	if err := json.Unmarshal([]byte(val), &embedding); err != nil {
		return nil, fmt.Errorf("unmarshal embedding: %w", err)
	}
	return embedding, nil
}

func (a *aiEmbeddingCacheAdapter) Set(ctx context.Context, key string, embedding []float64, ttl time.Duration) error {
	data, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}
	return a.client.Set(ctx, key, data, ttl).Err()
}

func (a *aiEmbeddingCacheAdapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}

func (a *aiEmbeddingCacheAdapter) GenerateKey(model string, input string) string {
	// Create a hash of model + input for the cache key
	h := sha256.New()
	h.Write([]byte(model))
	h.Write([]byte(":"))
	h.Write([]byte(input))
	hash := hex.EncodeToString(h.Sum(nil))
	return embeddingKeyPrefix + hash[:32] // Use first 32 chars of hash
}

// Compile-time check
var _ outbound.AIEmbeddingCachePort = (*aiEmbeddingCacheAdapter)(nil)
