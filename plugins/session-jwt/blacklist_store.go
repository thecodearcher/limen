package sessionjwt

import (
	"context"
	"fmt"
	"time"

	"github.com/thecodearcher/aegis"
)

type blacklistStore interface {
	Add(ctx context.Context, jti string, expiresAt time.Time) error
	Has(ctx context.Context, jti string) (bool, error)
	Prune(ctx context.Context) error
}

// cacheBlacklistStore stores blacklist entries in the shared CacheAdapter.
// TTL-based expiry means Prune is a no-op.
type cacheBlacklistStore struct {
	cache  aegis.CacheAdapter
	prefix string
}

func (s *cacheBlacklistStore) key(jti string) string {
	return fmt.Sprintf("%s:jwt:blacklist:%s", s.prefix, jti)
}

func (s *cacheBlacklistStore) Add(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := max(time.Until(expiresAt), 0)
	return s.cache.Set(ctx, s.key(jti), []byte("1"), ttl)
}

func (s *cacheBlacklistStore) Has(ctx context.Context, jti string) (bool, error) {
	return s.cache.Has(ctx, s.key(jti))
}

func (s *cacheBlacklistStore) Prune(_ context.Context) error {
	//no-op since TTL handles expiry
	return nil
}

// dbBlacklistStore stores blacklist entries in the jwt_blacklist database table.
type dbBlacklistStore struct {
	core   *aegis.AegisCore
	schema *blacklistSchema
}

func (s *dbBlacklistStore) Add(ctx context.Context, jti string, expiresAt time.Time) error {
	entry := &BlacklistEntry{
		JTI:       jti,
		ExpiresAt: expiresAt,
	}
	return s.core.Create(ctx, s.schema, entry, nil)
}

func (s *dbBlacklistStore) Has(ctx context.Context, jti string) (bool, error) {
	return s.core.Exists(ctx, s.schema, []aegis.Where{
		aegis.Eq(string(BlacklistSchemaJTIField), jti),
	})
}

func (s *dbBlacklistStore) Prune(ctx context.Context) error {
	return s.core.Delete(ctx, s.schema, []aegis.Where{
		aegis.Lt(s.schema.GetExpiresAtField(), time.Now()),
	})
}
