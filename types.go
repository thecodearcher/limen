package aegis

import (
	"context"
	"net/http"
	"time"

	"github.com/thecodearcher/aegis/schemas"
)

// this file contains the types for the aegis library
// NO FEATURES SHOULD BE ADDED TO THIS FILE those go in the feature.go file

// TokenGenerator defines the interface for JWT token generation and validation
type TokenGenerator interface {
	GenerateToken(claims map[string]interface{}, duration time.Duration) (string, error)
	GenerateAccessToken(sessionID string, user *schemas.User) (string, string, error)
	VerifyToken(token string) (map[string]any, error)
}

// AuthenticationResult represents the result of an authentication process and includes additional
type AuthenticationResult struct {
	// User represents the authenticated user
	User *User
	// Pending actions to be completed by the user before they can access the application e.g: two-factor authentication, email verification etc.
	PendingActions []PendingAction
}

// alias for the user model
type User = schemas.User
type SchemaConfig = schemas.Config

type UserSchema = schemas.UserSchema
type UserFields = schemas.UserFields
type VerificationSchema = schemas.VerificationSchema
type VerificationFields = schemas.VerificationFields

// Session alias for the session model
type Session = schemas.Session
type SessionSchema = schemas.SessionSchema
type SessionFields = schemas.SessionFields

// SessionResult contains the result of session operations
type SessionResult struct {
	Token        string // JWT token for JWT/hybrid strategies
	RefreshToken string // Refresh token if enabled
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
	Metadata map[string]interface{}
}

// SessionStore defines the interface for session storage backends
type SessionStore interface {
	// Create creates a new session with the given ID and data
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session *Session) error

	// Delete removes a session by ID
	Delete(ctx context.Context, sessionID string) error

	// DeleteByUserID removes all sessions for a specific user
	DeleteByUserID(ctx context.Context, userID string) error

	// Cleanup removes expired sessions
	Cleanup(ctx context.Context) error

	// Exists checks if a session exists
	Exists(ctx context.Context, sessionID string) (bool, error)

	// Count returns the number of active sessions for a user
	Count(ctx context.Context, userID string) (int, error)
}

// SessionStrategy defines the interface for different session management strategies
type SessionStrategy interface {
	// Create creates a new session for the given user
	Create(ctx context.Context, user *User) (*SessionResult, error)

	// Validate validates a session from the request
	Validate(ctx context.Context, request *http.Request) (*SessionValidateResult, error)

	// Refresh refreshes an existing session
	Refresh(ctx context.Context, request *http.Request) (*SessionRefreshResult, error)

	// GetName returns the strategy name
	GetName() string

	// IsStateful returns whether the strategy is stateful
	IsStateful() bool
}
