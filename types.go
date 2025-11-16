package aegis

import (
	"context"
	"net/http"
	"time"
)

// this file contains the types for the aegis library
// NO FEATURES SHOULD BE ADDED TO THIS FILE those go in the feature.go file

// TokenGenerator defines the interface for JWT token generation and validation
type TokenGenerator interface {
	// GenerateToken generates a JWT token with the given claims and duration
	GenerateToken(claims map[string]interface{}, duration time.Duration) (string, error)
	// GenerateAccessToken generates an access token with the configured duration and claims.
	// If duration is nil, the configured duration will be used. AdditionalClaims will be added to the claims.
	GenerateAccessToken(sessionID string, user *User, duration *time.Duration, additionalClaims map[string]any) (string, string, error)
	// VerifyToken verifies a JWT token and returns the claims
	VerifyToken(token string) (map[string]any, error)
}

// AuthenticationResult represents the result of an authentication process and includes additional
type AuthenticationResult struct {
	// User represents the authenticated user
	User *User
	// Pending actions to be completed by the user before they can access the application e.g: two-factor authentication, email verification etc.
	PendingActions []PendingAction
}

// SessionResult contains the result of session operations
type SessionResult struct {
	Token        string       `json:"token,omitzero"`         // Session token (cookie value, JWT token for JWT/hybrid strategies)
	RefreshToken string       `json:"refresh_token,omitzero"` // Refresh token if enabled
	Cookie       *http.Cookie `json:"-"`
}

type SessionRefreshResult struct {
	Token          string // JWT token for JWT/hybrid strategies
	RefreshToken   string // Refresh token if enabled
	UserID         any    // User ID
	ShouldStore    bool   // Whether the session should be stored in the database
	StaleSessionID string // The ID of the stale session that was replaced
}

type SessionValidateResult struct {
	UserID   any
	User     *User
	Session  *Session
	Metadata map[string]interface{}
}

// SessionStore defines the interface for session storage backends
type SessionStore interface {
	// Create creates a new session with the given ID and data
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by token
	Get(ctx context.Context, sessionToken string) (*Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session *Session) error

	// Delete removes a session by token
	Delete(ctx context.Context, sessionToken string) error

	// DeleteByUserID removes all sessions for a specific user
	DeleteByUserID(ctx context.Context, userID any) error
}

// SessionStrategy defines the interface for different session management strategies
type SessionStrategy interface {
	// Create creates a new session for the given user
	Create(ctx context.Context, user *User, temporaryAuth bool) (*SessionResult, error)

	// Validate validates a session from the request
	Validate(ctx context.Context, request *http.Request) (*SessionValidateResult, error)

	// Refresh refreshes an existing session
	Refresh(ctx context.Context, request *http.Request) (*SessionRefreshResult, error)

	// GetName returns the strategy name
	GetName() string

	// IsStateful returns whether the strategy is stateful
	IsStateful() bool
}
