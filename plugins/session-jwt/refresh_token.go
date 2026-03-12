package sessionjwt

import (
	"context"
	"fmt"
	"time"

	"github.com/thecodearcher/aegis"
)

// RefreshToken is the domain model stored in jwt_refresh_tokens.
type RefreshToken struct {
	ID        any
	Token     string
	UserID    any
	JWTID     string
	Family    string
	ExpiresAt time.Time
	CreatedAt time.Time
	raw       map[string]any
}

func (r *RefreshToken) Raw() map[string]any {
	return r.raw
}

// BlacklistEntry is the domain model stored in jwt_blacklist.
type BlacklistEntry struct {
	JTI       string
	ExpiresAt time.Time
	raw       map[string]any
}

func (b *BlacklistEntry) Raw() map[string]any {
	return b.raw
}

func (p *sessionJWTPlugin) CreateRefreshToken(ctx context.Context, userID any, jwtID string, family string, expiresAt *time.Time) (*RefreshToken, error) {
	now := time.Now()
	exp := now.Add(p.config.refreshTokenDuration)
	if expiresAt != nil {
		exp = *expiresAt
	}
	rt := &RefreshToken{
		Token:     generateOpaqueToken(),
		UserID:    userID,
		JWTID:     jwtID,
		Family:    family,
		ExpiresAt: exp,
		CreatedAt: now,
	}
	if err := p.core.Create(ctx, p.refreshTokenSchema, rt, nil); err != nil {
		return nil, fmt.Errorf("session-jwt: failed to store refresh token: %w", err)
	}
	return rt, nil
}

func (p *sessionJWTPlugin) FindRefreshTokenByToken(ctx context.Context, token string) (*RefreshToken, error) {
	model, err := p.core.FindOne(ctx, p.refreshTokenSchema, []aegis.Where{
		aegis.Eq(p.refreshTokenSchema.GetTokenField(), token),
	}, nil)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	return model.(*RefreshToken), nil
}

func (p *sessionJWTPlugin) DeleteRefreshToken(ctx context.Context, token string) error {
	return p.core.Delete(ctx, p.refreshTokenSchema, []aegis.Where{
		aegis.Eq(p.refreshTokenSchema.GetTokenField(), token),
	})
}

func (p *sessionJWTPlugin) DeleteRefreshTokenFamily(ctx context.Context, family string) error {
	return p.core.Delete(ctx, p.refreshTokenSchema, []aegis.Where{
		aegis.Eq(p.refreshTokenSchema.GetFamilyField(), family),
	})
}

func (p *sessionJWTPlugin) DeleteRefreshTokensByUserID(ctx context.Context, userID any) error {
	return p.core.Delete(ctx, p.refreshTokenSchema, []aegis.Where{
		aegis.Eq(p.refreshTokenSchema.GetUserIDField(), userID),
	})
}

func (p *sessionJWTPlugin) FamilyHasActiveTokens(ctx context.Context, family string) (bool, error) {
	return p.core.Exists(ctx, p.refreshTokenSchema, []aegis.Where{
		aegis.Eq(p.refreshTokenSchema.GetFamilyField(), family),
	})
}

// RotateRefreshToken deletes the old refresh token and creates a new one in
// the same family. Returns the new refresh token.
func (p *sessionJWTPlugin) RotateRefreshToken(ctx context.Context, old *RefreshToken, newJWTID string) (*RefreshToken, error) {
	if err := p.DeleteRefreshToken(ctx, old.Token); err != nil {
		return nil, fmt.Errorf("session-jwt: failed to delete old refresh token: %w", err)
	}
	return p.CreateRefreshToken(ctx, old.UserID, newJWTID, old.Family, &old.ExpiresAt)
}

// AddToBlacklist records a revoked JWT so ValidateSession can reject it.
func (p *sessionJWTPlugin) AddToBlacklist(ctx context.Context, jti string, expiresAt time.Time) error {
	if !p.config.blacklistEnabled {
		return nil
	}
	return p.blacklist.Add(ctx, jti, expiresAt)
}

// IsBlacklisted checks whether a JWT ID has been revoked.
func (p *sessionJWTPlugin) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if !p.config.blacklistEnabled {
		return false, nil
	}
	return p.blacklist.Has(ctx, jti)
}

// PruneExpiredBlacklist removes blacklist entries whose JWT has already
// expired naturally, keeping the table compact. No-op when using cache (TTL handles expiry).
func (p *sessionJWTPlugin) PruneExpiredBlacklist(ctx context.Context) error {
	if !p.config.blacklistEnabled {
		return nil
	}
	return p.blacklist.Prune(ctx)
}

// PruneExpiredRefreshTokens removes refresh tokens that have passed their
// absolute expiry, keeping the table compact.
func (p *sessionJWTPlugin) PruneExpiredRefreshTokens(ctx context.Context) error {
	if !p.config.refreshTokenEnabled {
		return nil
	}
	return p.core.Delete(ctx, p.refreshTokenSchema, []aegis.Where{
		aegis.Lt(p.refreshTokenSchema.GetExpiresAtField(), time.Now()),
	})
}
