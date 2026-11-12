package limen

import (
	"context"
	"net/http"
	"time"
)

// this file contains the types for the limen library
// NO FEATURES SHOULD BE ADDED TO THIS FILE those go in the plugin.go file

// OAuthAccountProfile holds the data returned by the provider after a successful OAuth authentication.
type OAuthAccountProfile struct {
	Provider             string
	ProviderAccountID    string
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt *time.Time
	Scope                string
	IDToken              string
	Email                string
	EmailVerified        bool
	Name                 string
	AvatarURL            string
	Raw                  map[string]any
}

type SessionCreateOptions struct {
	ShortSession bool
	// ExpiresIn overrides the session lifetime when positive.
	ExpiresIn time.Duration
	// Data holds values for registered session fields. SessionValue is accepted.
	Data map[SchemaField]any
}

type SessionCreateOption func(*SessionCreateOptions)

// WithShortSession sets the short session flag for the session.
func WithShortSession(shortSession bool) SessionCreateOption {
	return func(o *SessionCreateOptions) {
		o.ShortSession = shortSession
	}
}

// WithSessionExpiresIn sets a fixed lifetime for the session.
func WithSessionExpiresIn(expiresIn time.Duration) SessionCreateOption {
	return func(o *SessionCreateOptions) {
		o.ExpiresIn = expiresIn
	}
}

// WithSessionData sets registered session fields at creation.
func WithSessionData(data map[SchemaField]any) SessionCreateOption {
	return func(o *SessionCreateOptions) {
		o.Data = data
	}
}

// ResolveSessionCreateOptions resolves options for SessionManager implementations.
func ResolveSessionCreateOptions(opts ...SessionCreateOption) SessionCreateOptions {
	var o SessionCreateOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// SessionManager defines the interface for session lifecycle management.
type SessionManager interface {
	CreateSession(ctx context.Context, r *http.Request, auth *AuthenticationResult, opts ...SessionCreateOption) (*SessionResult, error)
	ValidateSession(ctx context.Context, r *http.Request) (*ValidatedSession, error)
	RevokeSession(ctx context.Context, token string) error
	RevokeAllSessions(ctx context.Context, userID any) error
	ListSessions(ctx context.Context, userID any) ([]Session, error)

	// GetSessionData returns the value stored on the session under the logical field.
	GetSessionData(ctx context.Context, session *Session, field SchemaField) (any, error)
	// UpdateSession persists the fields; the live session reflects the new state on
	// return. A non-nil SessionResult carries a re-issued token that must be
	// delivered to the client; nil means the current token stays valid.
	UpdateSession(ctx context.Context, session *Session, data map[SchemaField]any) (*SessionResult, error)
	// UpdateSessions persists the fields on every session whose stored values equal
	// match, where the backing store supports it — see SessionBulkUpdater.
	UpdateSessions(ctx context.Context, data map[SchemaField]any, match map[SchemaField]any) error
}

// SessionValue carries both representations of a session-data value so each
// session manager stores the one matching its exposure: Internal goes to
// server-side storage, Client to client-readable tokens. Plain (unwrapped)
// values are stored as-is by every manager.
type SessionValue struct {
	Internal any
	Client   any
}

// ValidatedSession is the result of a session validation.
type ValidatedSession struct {
	User      *User
	Session   *Session
	Refreshed *SessionResult // Set if session was extended during validation
}

// AuthenticationResult represents the result of an authentication process.
type AuthenticationResult struct {
	User *User
}

// SessionResult contains token and delivery information for a session.
type SessionResult struct {
	Token        string       `json:"token,omitzero"`
	RefreshToken string       `json:"refreshToken,omitzero"`
	Cookie       *http.Cookie `json:"-"`
	// ShortSession indicates if the session is a short session i.e expires in less than the global session duration
	// This is typically when "remember me" is not checked.
	ShortSession *bool
	// ExtraCookies holds additional cookies that session managers or plugins need to
	// deliver alongside the main session cookie (e.g. refresh tokens).
	ExtraCookies []*http.Cookie `json:"-"`
	// Session is the session the result was issued for, when the session manager provides it.
	Session *Session `json:"-"`
}

// SessionStore defines the interface for session storage backends.
type SessionStore interface {
	Get(ctx context.Context, token string) (*Session, error)
	// Set persists session state, upserted by token. It accepts a *Session or a
	// map[SchemaField]any row subset, which must include SessionSchemaTokenField.
	Set(ctx context.Context, data any) error
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID any) error
	ListByUserID(ctx context.Context, userID any) ([]Session, error)
}

// SessionBulkUpdater is an optional SessionStore capability for updating every
// session whose stored values equal match. Stores without it treat bulk updates
// as a no-op; consumers must not rely on session data for revocation.
type SessionBulkUpdater interface {
	UpdateSessions(ctx context.Context, data map[SchemaField]any, match map[SchemaField]any) error
}

// RateLimiterStore defines the interface for rate-limit storage backends.
type RateLimiterStore interface {
	Get(ctx context.Context, key string) (*RateLimit, error)
	Set(ctx context.Context, key string, value *RateLimit, ttl time.Duration) error
}

// IDGenerator generates IDs for database records.
type IDGenerator interface {
	Generate(ctx context.Context) (any, error)
	GetColumnType() ColumnType
}
