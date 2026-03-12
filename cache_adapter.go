package aegis

import (
	"context"
	"time"
)

// CacheAdapter is a key-value store with TTL support used by sessions,
// rate limiting, JWT blacklists, and other features that benefit from
// a shared cache backend.
//
// The default implementation is MemoryCacheStore (in-process maps).
// For multi-instance deployments, plug in a Redis-backed implementation
// via Config.CacheStore.
type CacheAdapter interface {
	// Get retrieves the value associated with key.
	// Returns ErrRecordNotFound if the key does not exist or has expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores value under key with the given TTL.
	// A TTL of 0 means the entry never expires.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Has reports whether key exists and has not expired.
	Has(ctx context.Context, key string) (bool, error)

	// Delete removes the entry for key. It is a no-op if key does not exist.
	Delete(ctx context.Context, key string) error
}
