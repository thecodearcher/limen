package sessionjwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/thecodearcher/limen"
)

type ConfigOption func(*config)

type config struct {
	signingMethod        jwt.SigningMethod
	signingKey           any
	verificationKey      any
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	refreshTokenRotation bool
	customClaims         func(user *limen.User) map[string]any
	issuer               string
	audience             []string
	blacklistEnabled     bool
	blacklistStoreType   limen.StoreType
	refreshTokenEnabled  bool
	subjectEncoder       func(user *limen.User) string
	subjectResolver      func(subject string) (any, error)
	refreshUser          bool
}

// WithSigningMethod sets the JWT signing algorithm (default: HS256).
func WithSigningMethod(method jwt.SigningMethod) ConfigOption {
	return func(c *config) {
		c.signingMethod = method
	}
}

// WithSigningKey sets the key used to sign JWTs.
//
// Accepted types per signing method family:
//
//	HMAC  (HS256, …)  — string. Falls back to Config.SigningSecret when unset.
//	RSA   (RS256, …)  — PEM string or *rsa.PrivateKey.
//	ECDSA (ES256, …)  — PEM string or *ecdsa.PrivateKey.
//	EdDSA             — PEM string or ed25519.PrivateKey.
//
// PEM strings are automatically parsed during initialization.
func WithSigningKey(key any) ConfigOption {
	return func(c *config) {
		c.signingKey = key
	}
}

// WithVerificationKey sets the key used to verify JWTs. Only needed for
// asymmetric algorithms when the verification key differs from the signing
// key (e.g. a separate public key endpoint). If omitted, the public key is
// automatically derived from the private signing key.
//
// Accepted types per signing method family:
//
//	RSA   — PEM string or *rsa.PublicKey.
//	ECDSA — PEM string or *ecdsa.PublicKey.
//	EdDSA — PEM string or ed25519.PublicKey.
//
// Not applicable for HMAC (the signing key is used for both operations).
func WithVerificationKey(key any) ConfigOption {
	return func(c *config) {
		c.verificationKey = key
	}
}

// WithAccessTokenDuration sets how long JWTs are valid (default: 15m).
func WithAccessTokenDuration(d time.Duration) ConfigOption {
	return func(c *config) {
		c.accessTokenDuration = d
	}
}

// WithRefreshTokenDuration sets how long refresh tokens are valid (default: 7d).
func WithRefreshTokenDuration(d time.Duration) ConfigOption {
	return func(c *config) {
		c.refreshTokenDuration = d
	}
}

// WithRefreshTokenRotation toggles refresh-token rotation (default: true).
// When enabled, each use of a refresh token issues a new one and invalidates
// the previous token.
func WithRefreshTokenRotation(enabled bool) ConfigOption {
	return func(c *config) {
		c.refreshTokenRotation = enabled
	}
}

// WithCustomClaims registers a function that adds extra claims to every JWT.
func WithCustomClaims(fn func(user *limen.User) map[string]any) ConfigOption {
	return func(c *config) {
		c.customClaims = fn
	}
}

// WithIssuer sets the "iss" claim on every JWT.
func WithIssuer(issuer string) ConfigOption {
	return func(c *config) {
		c.issuer = issuer
	}
}

// WithAudience sets the "aud" claim on every JWT.
func WithAudience(audience []string) ConfigOption {
	return func(c *config) {
		c.audience = audience
	}
}

// WithBlacklistEnabled enables an optional JWT blacklist. When a session is revoked
// the JWT's jti is recorded so that ValidateSession can reject it before its
// natural expiry. This adds a cache or DB lookup on every validation call depending on the store type.
func WithBlacklistEnabled(enabled bool) ConfigOption {
	return func(c *config) {
		c.blacklistEnabled = enabled
	}
}

// WithBlacklistStoreType selects the storage backend for the JWT blacklist.
// When set to StoreTypeCache, entries are stored in the shared CacheAdapter
// with a TTL equal to the token's remaining lifetime.
// Defaults to StoreTypeCache.
func WithBlacklistStoreType(storeType limen.StoreType) ConfigOption {
	return func(c *config) {
		c.blacklistStoreType = storeType
	}
}

// WithRefreshToken enables or disables refresh tokens (default: true).
func WithRefreshToken(enabled bool) ConfigOption {
	return func(c *config) {
		c.refreshTokenEnabled = enabled
	}
}

// WithSubject sets a custom function to derive the JWT "sub" claim from a
// user. By default the raw user.ID is used, which may expose internal
// database identifiers. Use this to substitute a public UUID or other
// opaque value.
func WithSubject(fn func(user *limen.User) string) ConfigOption {
	return func(c *config) {
		c.subjectEncoder = fn
	}
}

// WithSubjectResolver sets a function that converts a JWT "sub" claim
// back to a user ID value or Return limen.User which will be used to populate the User field on the ValidatedSession.
// Default: returns the subject string as-is.
func WithSubjectResolver(fn func(subject string) (any, error)) ConfigOption {
	return func(c *config) {
		c.subjectResolver = fn
	}
}

// WithRefreshUser controls whether ValidateSession fetches a fresh user
// from the database after verifying the JWT (default: false).
// When enabled, user-specific fields (email, etc.) are omitted from the
// JWT to reduce token size, and the full user is loaded from the DB on
// every validation call.
func WithRefreshUser(enabled bool) ConfigOption {
	return func(c *config) {
		c.refreshUser = enabled
	}
}
