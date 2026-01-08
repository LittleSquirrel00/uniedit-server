package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/port/outbound"
)

const (
	oauthStateKeyPrefix = "oauth:state:"
	oauthStateTTL       = 10 * time.Minute
)

// oauthStateStore implements outbound.OAuthStateStorePort.
type oauthStateStore struct {
	client *redis.Client
}

// NewOAuthStateStore creates a new OAuth state store adapter.
func NewOAuthStateStore(client *redis.Client) outbound.OAuthStateStorePort {
	return &oauthStateStore{client: client}
}

func (s *oauthStateStore) Set(ctx context.Context, state string, provider string) error {
	key := oauthStateKeyPrefix + state
	return s.client.Set(ctx, key, provider, oauthStateTTL).Err()
}

func (s *oauthStateStore) Get(ctx context.Context, state string) (string, error) {
	key := oauthStateKeyPrefix + state
	provider, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("state not found")
		}
		return "", err
	}
	return provider, nil
}

func (s *oauthStateStore) Delete(ctx context.Context, state string) error {
	key := oauthStateKeyPrefix + state
	return s.client.Del(ctx, key).Err()
}

// Compile-time check
var _ outbound.OAuthStateStorePort = (*oauthStateStore)(nil)
