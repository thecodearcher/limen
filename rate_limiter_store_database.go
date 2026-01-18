package aegis

import "context"

type DatabaseRateLimiterStore struct {
	core *AegisCore
}

func NewDatabaseRateLimiterStore(core *AegisCore) RateLimiterStore {
	return &DatabaseRateLimiterStore{core: core}
}

func (s *DatabaseRateLimiterStore) Get(ctx context.Context, key string) (*RateLimit, error) {
	limit, err := s.core.FindOne(ctx, s.core.Schema.RateLimit, []Where{
		Eq(s.core.Schema.RateLimit.GetKeyField(), key),
	}, nil)

	if err != nil {
		return nil, ErrRateLimitNotFound
	}

	return limit.(*RateLimit), nil
}

func (s *DatabaseRateLimiterStore) Create(ctx context.Context, value *RateLimit) error {
	return s.core.Create(ctx, s.core.Schema.RateLimit, value, nil)
}

func (d *DatabaseRateLimiterStore) Update(ctx context.Context, key string, value *RateLimit) error {
	return d.core.UpdateRaw(ctx, d.core.Schema.RateLimit, value, []Where{
		Eq(d.core.Schema.RateLimit.GetKeyField(), key),
	}, false)
}
