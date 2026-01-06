package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// EmbeddingCache provides caching for embeddings.
type EmbeddingCache struct {
	client     redis.UniversalClient
	prefix     string
	ttl        time.Duration
	maxEntries int

	// Stats tracking
	statsPrefix string
}

// EmbeddingCacheConfig contains cache configuration.
type EmbeddingCacheConfig struct {
	Prefix      string
	StatsPrefix string
	TTL         time.Duration
	MaxEntries  int
}

// DefaultEmbeddingCacheConfig returns the default cache configuration.
func DefaultEmbeddingCacheConfig() *EmbeddingCacheConfig {
	return &EmbeddingCacheConfig{
		Prefix:      "emb:",
		StatsPrefix: "emb_stats:",
		TTL:         24 * time.Hour,
		MaxEntries:  100000,
	}
}

// NewEmbeddingCache creates a new embedding cache.
func NewEmbeddingCache(client redis.UniversalClient, config *EmbeddingCacheConfig) *EmbeddingCache {
	if config == nil {
		config = DefaultEmbeddingCacheConfig()
	}

	return &EmbeddingCache{
		client:      client,
		prefix:      config.Prefix,
		statsPrefix: config.StatsPrefix,
		ttl:         config.TTL,
		maxEntries:  config.MaxEntries,
	}
}

// CachedEmbedding represents a cached embedding.
type CachedEmbedding struct {
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
	CreatedAt int64     `json:"created_at"`
}

// Get retrieves an embedding from cache.
func (c *EmbeddingCache) Get(ctx context.Context, model, text string) (*CachedEmbedding, error) {
	key := c.makeKey(model, text)

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("get from cache: %w", err)
	}

	var cached CachedEmbedding
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("unmarshal cached embedding: %w", err)
	}

	return &cached, nil
}

// Set stores an embedding in cache.
func (c *EmbeddingCache) Set(ctx context.Context, model, text string, embedding []float32) error {
	key := c.makeKey(model, text)

	cached := &CachedEmbedding{
		Embedding: embedding,
		Model:     model,
		CreatedAt: time.Now().Unix(),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("set in cache: %w", err)
	}

	return nil
}

// GetBatch retrieves multiple embeddings from cache.
// Returns a map of text -> embedding for cache hits, and a slice of texts for cache misses.
func (c *EmbeddingCache) GetBatch(ctx context.Context, model string, texts []string) (map[string]*CachedEmbedding, []string, error) {
	if len(texts) == 0 {
		return nil, nil, nil
	}

	// Build keys
	keys := make([]string, len(texts))
	keyToText := make(map[string]string, len(texts))
	for i, text := range texts {
		key := c.makeKey(model, text)
		keys[i] = key
		keyToText[key] = text
	}

	// Get all at once
	results, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, texts, fmt.Errorf("mget from cache: %w", err)
	}

	hits := make(map[string]*CachedEmbedding)
	var misses []string

	for i, result := range results {
		text := keyToText[keys[i]]
		if result == nil {
			misses = append(misses, text)
			continue
		}

		data, ok := result.(string)
		if !ok {
			misses = append(misses, text)
			continue
		}

		var cached CachedEmbedding
		if err := json.Unmarshal([]byte(data), &cached); err != nil {
			misses = append(misses, text)
			continue
		}

		hits[text] = &cached
	}

	return hits, misses, nil
}

// SetBatch stores multiple embeddings in cache.
func (c *EmbeddingCache) SetBatch(ctx context.Context, model string, embeddings map[string][]float32) error {
	if len(embeddings) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	now := time.Now().Unix()

	for text, embedding := range embeddings {
		key := c.makeKey(model, text)
		cached := &CachedEmbedding{
			Embedding: embedding,
			Model:     model,
			CreatedAt: now,
		}

		data, err := json.Marshal(cached)
		if err != nil {
			continue
		}

		pipe.Set(ctx, key, data, c.ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline exec: %w", err)
	}

	return nil
}

// Delete removes an embedding from cache.
func (c *EmbeddingCache) Delete(ctx context.Context, model, text string) error {
	key := c.makeKey(model, text)
	return c.client.Del(ctx, key).Err()
}

// Clear removes all embeddings from cache.
func (c *EmbeddingCache) Clear(ctx context.Context) error {
	pattern := c.prefix + "*"

	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan keys: %w", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete keys: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Stats returns cache statistics.
func (c *EmbeddingCache) Stats(ctx context.Context) (*CacheStats, error) {
	pattern := c.prefix + "*"

	var count int64
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan keys: %w", err)
		}

		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return &CacheStats{
		EntryCount: count,
		MaxEntries: int64(c.maxEntries),
		TTL:        c.ttl,
	}, nil
}

// CacheStats represents cache statistics.
type CacheStats struct {
	EntryCount int64         `json:"entry_count"`
	MaxEntries int64         `json:"max_entries"`
	TTL        time.Duration `json:"ttl"`
}

// APIKeyCacheStats represents cache hit/miss stats for an API key.
type APIKeyCacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
}

// RecordHit records a cache hit for an API key.
func (c *EmbeddingCache) RecordHit(ctx context.Context, apiKeyID string) error {
	key := c.statsPrefix + "hits:" + apiKeyID
	return c.client.Incr(ctx, key).Err()
}

// RecordMiss records a cache miss for an API key.
func (c *EmbeddingCache) RecordMiss(ctx context.Context, apiKeyID string) error {
	key := c.statsPrefix + "misses:" + apiKeyID
	return c.client.Incr(ctx, key).Err()
}

// RecordBatchStats records cache hits and misses for an API key.
func (c *EmbeddingCache) RecordBatchStats(ctx context.Context, apiKeyID string, hits, misses int64) error {
	if hits == 0 && misses == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	if hits > 0 {
		key := c.statsPrefix + "hits:" + apiKeyID
		pipe.IncrBy(ctx, key, hits)
	}
	if misses > 0 {
		key := c.statsPrefix + "misses:" + apiKeyID
		pipe.IncrBy(ctx, key, misses)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// GetAPIKeyStats returns cache statistics for an API key.
func (c *EmbeddingCache) GetAPIKeyStats(ctx context.Context, apiKeyID string) (*APIKeyCacheStats, error) {
	hitsKey := c.statsPrefix + "hits:" + apiKeyID
	missesKey := c.statsPrefix + "misses:" + apiKeyID

	results, err := c.client.MGet(ctx, hitsKey, missesKey).Result()
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	var hits, misses int64
	if results[0] != nil {
		if v, ok := results[0].(string); ok {
			fmt.Sscanf(v, "%d", &hits)
		}
	}
	if results[1] != nil {
		if v, ok := results[1].(string); ok {
			fmt.Sscanf(v, "%d", &misses)
		}
	}

	return &APIKeyCacheStats{
		Hits:   hits,
		Misses: misses,
	}, nil
}

// ResetAPIKeyStats resets cache statistics for an API key.
func (c *EmbeddingCache) ResetAPIKeyStats(ctx context.Context, apiKeyID string) error {
	hitsKey := c.statsPrefix + "hits:" + apiKeyID
	missesKey := c.statsPrefix + "misses:" + apiKeyID
	return c.client.Del(ctx, hitsKey, missesKey).Err()
}

// makeKey creates a cache key for an embedding.
func (c *EmbeddingCache) makeKey(model, text string) string {
	// Hash the text to create a fixed-length key
	hash := sha256.Sum256([]byte(text))
	return c.prefix + model + ":" + hex.EncodeToString(hash[:])
}
