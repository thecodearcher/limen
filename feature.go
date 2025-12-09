package aegis

import (
	"context"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

// This file contains the interfaces for the features of the aegis library.
// and serves as a contract for the features of the library.
// Ensures that the features are implemented correctly in their respective modules.

type FeatureName string

const (
	FeatureEmailPassword FeatureName = "email-password"
)

type Feature interface {
	// Unique identifier for the feature.
	Name() FeatureName
	// Initialize initializes the feature.
	Initialize(core *AegisCore) error
	// PluginHTTPConfig returns the configuration for the plugin's HTTP surface.
	PluginHTTPConfig() PluginHTTPConfig
	// RegisterRoutes registers routes for the plugin.
	RegisterRoutes(httpCore *AegisHTTPCore, routeBuilder *RouteBuilder)
	// GetSchemas returns all schemas provided by this feature.
	// Returns a map of schema name to SchemaIntrospector.
	// Plugins can extend core schemas by setting Extends field, or create new tables.
	// If a plugin extends a core schema, it should return a schema with the same name
	// and set Extends to the core schema name (e.g., "users").
	GetSchemas() []SchemaIntrospector
}

// PluginHTTPConfig is the configuration for the plugin's HTTP surface.
//
// Note: The base path is relative to the Aegis base path and can be overridden by the end user.
type PluginHTTPConfig struct {
	// The base path where the plugin's routes will be mounted.
	BasePath string
	// Middleware to be applied to the plugin's routes.
	Middleware []httpx.Middleware
	// Specific rate limit rules to be applied to the plugin's routes.
	// These rules can be overridden by the end user.
	RateLimitRules []*RateLimitRule
}

type EmailPasswordFeature interface {
	// SignInWithEmailAndPassword signs in a user with the given email and password
	// and returns the authentication result.
	SignInWithEmailAndPassword(ctx context.Context, email string, password string) (*AuthenticationResult, error)

	// SignUpWithEmailAndPassword creates a new user with the given email and password
	// and returns the authentication result.
	// additionalFields are additional fields to be added to the user.
	//
	// Note: When a key in additionalFields is already present in User, the value in additionalFields will be overwritten by the value associated with the key in User.
	SignUpWithEmailAndPassword(ctx context.Context, user *User, additionalFields map[string]any) (*AuthenticationResult, error)

	// HashPassword hashes the given password and returns the hash.
	// This is used to hash the password before storing it in the database.
	//
	// You only call this if you are not using the SignUpWithEmailAndPassword method.
	HashPassword(password string) (string, error)

	// ComparePassword compares the given password with the given hash and returns true if they match, false otherwise.
	ComparePassword(password string, hash string) (bool, error)

	// RequestPasswordReset requests a password reset for the given email.
	// Returns the verification object if the request is successful.
	RequestPasswordReset(ctx context.Context, email string) (*Verification, error)

	// ResetPassword resets the password using the given token and new password.
	ResetPassword(ctx context.Context, token string, newPassword string) error

	// UpdatePassword updates the password for the given user.
	//
	// Note: If revokeOtherSessions is true, the current session will be revoked and a new session should be created.
	UpdatePassword(ctx context.Context, user *User, currentPassword string, newPassword string, revokeOtherSessions bool) error

	// RequestEmailVerification requests an email verification for the given user.
	RequestEmailVerification(ctx context.Context, user *User) (*Verification, error)

	// VerifyEmail verifies the email using the given token.
	VerifyEmail(ctx context.Context, token string) error
}
