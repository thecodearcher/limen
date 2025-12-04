package aegis

import (
	"context"
	"sync"
)

type MemoryRateLimiterStore struct {
	limits map[string]*RateLimit
	mu     sync.RWMutex
}

func NewMemoryRateLimiterStore() *MemoryRateLimiterStore {
	return &MemoryRateLimiterStore{
		limits: make(map[string]*RateLimit),
	}
}

func (s *MemoryRateLimiterStore) Get(ctx context.Context, key string) (*RateLimit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit, exists := s.limits[key]
	if !exists {
		return nil, ErrRateLimitNotFound
	}

	return limit, nil
}

func (s *MemoryRateLimiterStore) Create(ctx context.Context, value *RateLimit) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.limits[value.Key] = value
	return nil
}

func (s *MemoryRateLimiterStore) Update(ctx context.Context, key string, value *RateLimit) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.limits[key] = value
	return nil
}
