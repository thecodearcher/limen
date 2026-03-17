package limen

import (
	"context"
	"time"
)

type databaseRateLimiterStore struct {
	core *LimenCore
}

func newDatabaseRateLimiterStore(core *LimenCore) RateLimiterStore {
	return &databaseRateLimiterStore{core: core}
}

func (s *databaseRateLimiterStore) Get(ctx context.Context, key string) (*RateLimit, error) {
	limit, err := s.core.FindOne(ctx, s.core.Schema.RateLimit, []Where{
		Eq(s.core.Schema.RateLimit.GetKeyField(), key),
	}, nil)

	if err != nil {
		return nil, ErrRateLimitNotFound
	}

	return limit.(*RateLimit), nil
}

func (s *databaseRateLimiterStore) Set(ctx context.Context, key string, value *RateLimit, _ time.Duration) error {
	if value.ID == nil {
		return s.core.Create(ctx, s.core.Schema.RateLimit, value, nil)
	}
	return s.core.UpdateRaw(ctx, s.core.Schema.RateLimit, value, []Where{
		Eq(s.core.Schema.RateLimit.GetKeyField(), key),
	}, false)
}
