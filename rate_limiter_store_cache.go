package aegis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type cacheRateLimiterStore struct {
	cache  CacheAdapter
	prefix string
}

func newRateLimiterCacheStore(core *AegisCore) *cacheRateLimiterStore {
	return &cacheRateLimiterStore{
		cache:  core.CacheStore(),
		prefix: core.CacheKeyPrefix(),
	}
}

func (s *cacheRateLimiterStore) rateLimitKey(key string) string {
	return fmt.Sprintf("%s:rl:%s", s.prefix, key)
}

func (s *cacheRateLimiterStore) Get(ctx context.Context, key string) (*RateLimit, error) {
	data, err := s.cache.Get(ctx, s.rateLimitKey(key))
	if err != nil {
		return nil, ErrRateLimitNotFound
	}

	var rl RateLimit
	if err := json.Unmarshal(data, &rl); err != nil {
		return nil, ErrRateLimitNotFound
	}

	return &rl, nil
}

func (s *cacheRateLimiterStore) Set(ctx context.Context, key string, value *RateLimit, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal rate limit: %w", err)
	}

	return s.cache.Set(ctx, s.rateLimitKey(key), data, ttl)
}
