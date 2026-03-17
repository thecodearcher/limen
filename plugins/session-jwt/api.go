package sessionjwt

import (
	"context"
	"net/http"
	"time"

	"github.com/thecodearcher/limen"
)

// API is the public programmatic interface for the session-jwt plugin.
type API interface {
	// GenerateAccessToken creates a signed JWT for the given user and
	// returns the token string along with its unique JTI.
	GenerateAccessToken(user *limen.User) (token string, jti string, err error)

	// VerifyAccessToken parses and validates a JWT string, returning the
	// decoded claims on success.
	VerifyAccessToken(tokenString string) (*LimenClaims, error)

	// RefreshAccessToken extracts the refresh token from the request body,
	// validates it, and returns a new session result with a fresh JWT and
	// (if rotation is enabled) a new refresh token.
	RefreshAccessToken(r *http.Request) (*limen.SessionResult, *limen.User, error)

	// CreateRefreshToken stores a new opaque refresh token linked to the
	// given user and JWT ID. Pass nil for expiresAt to use the configured
	// duration, or a non-nil pointer to inherit an existing family expiry.
	CreateRefreshToken(ctx context.Context, userID any, jwtID string, family string, expiresAt *time.Time) (*RefreshToken, error)

	// FindRefreshTokenByToken looks up a refresh token by its opaque value.
	FindRefreshTokenByToken(ctx context.Context, token string) (*RefreshToken, error)

	// RotateRefreshToken atomically deletes the old refresh token and
	// creates a new one in the same family, preserving the family expiry.
	RotateRefreshToken(ctx context.Context, old *RefreshToken, newJWTID string) (*RefreshToken, error)

	// DeleteRefreshToken removes a single refresh token by its opaque value.
	DeleteRefreshToken(ctx context.Context, token string) error

	// DeleteRefreshTokenFamily removes all refresh tokens in a family,
	// used for reuse detection.
	DeleteRefreshTokenFamily(ctx context.Context, family string) error

	// DeleteRefreshTokensByUserID removes all refresh tokens for a user.
	DeleteRefreshTokensByUserID(ctx context.Context, userID any) error

	// FamilyHasActiveTokens reports whether any refresh tokens exist in
	// the given family.
	FamilyHasActiveTokens(ctx context.Context, family string) (bool, error)

	// AddToBlacklist revokes a specific JWT by adding its JTI to the
	// blacklist until the token's original expiry time.
	AddToBlacklist(ctx context.Context, jti string, expiresAt time.Time) error

	// IsBlacklisted reports whether a JWT with the given JTI has been
	// revoked.
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	// PruneExpiredBlacklist removes blacklist entries whose JWT has already
	// expired naturally. Call periodically (e.g. via cron) to keep the
	// blacklist table compact.
	PruneExpiredBlacklist(ctx context.Context) error

	// PruneExpiredRefreshTokens removes refresh tokens that have passed
	// their absolute expiry. Call periodically to keep the table compact.
	PruneExpiredRefreshTokens(ctx context.Context) error
}

// Use returns a type-safe API for the session-jwt plugin.
// Panics if the plugin was not registered in Config.Plugins.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginSessionJWT)
}
