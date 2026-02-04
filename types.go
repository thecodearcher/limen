package aegis

import (
	"context"
	"net/http"
	"time"
)

// this file contains the types for the aegis library
// NO FEATURES SHOULD BE ADDED TO THIS FILE those go in the feature.go file

// SessionManager defines the interface for session lifecycle management.
type SessionManager interface {
	CreateSession(ctx context.Context, r *http.Request, auth *AuthenticationResult) (*SessionResult, error)
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
	Token               string              `json:"token,omitzero"`
	Cookie              *http.Cookie        `json:"-"`
	TokenDeliveryMethod TokenDeliveryMethod `json:"-"`
}

// SessionUpdates contains fields for partial session updates.
type SessionUpdates struct {
	ExpiresAt  *time.Time
	LastAccess *time.Time
	Metadata   map[string]any
}

// SessionStore defines the interface for session storage backends.
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	GetByToken(ctx context.Context, token string) (*Session, error)
	UpdateByToken(ctx context.Context, token string, updates *SessionUpdates) error
	DeleteByToken(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID any) error
}

type RateLimiterStore interface {
	Get(ctx context.Context, key string) (*RateLimit, error)
	Create(ctx context.Context, value *RateLimit) error
	Update(ctx context.Context, key string, value *RateLimit) error
}

// IDGenerator generates IDs for database records.
type IDGenerator interface {
	Generate(ctx context.Context) (any, error)
	GetColumnType() ColumnType
}
