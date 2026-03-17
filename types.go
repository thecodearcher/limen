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
}

type SessionCreateOption func(*SessionCreateOptions)

// WithShortSession sets the short session flag for the session.
func WithShortSession(shortSession bool) SessionCreateOption {
	return func(o *SessionCreateOptions) {
		o.ShortSession = shortSession
	}
}

// SessionManager defines the interface for session lifecycle management.
type SessionManager interface {
	CreateSession(ctx context.Context, r *http.Request, auth *AuthenticationResult, shortSession bool) (*SessionResult, error)
	ValidateSession(ctx context.Context, r *http.Request) (*ValidatedSession, error)
	RevokeSession(ctx context.Context, token string) error
	RevokeAllSessions(ctx context.Context, userID any) error
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
}

// SessionStore defines the interface for session storage backends.
type SessionStore interface {
	Get(ctx context.Context, token string) (*Session, error)
	Set(ctx context.Context, session *Session) error
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID any) error
	ListByUserID(ctx context.Context, userID any) ([]Session, error)
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
