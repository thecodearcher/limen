package aegis

import (
	"context"

	"github.com/thecodearcher/aegis/pkg/httpx"
	"github.com/thecodearcher/aegis/schemas"
)

// This file contains the interfaces for the features of the aegis library.
// and serves as a contract for the features of the library.
// Ensures that the features are implemented correctly in their respective modules.

type FeatureName string

const (
	FeatureEmailPassword FeatureName = "email-password"
)

type Feature interface {
	Name() FeatureName
	Initialize(core *AegisCore) error
	HTTPMount(aegis *Aegis, responder *Responder, config *HTTPConfig) HTTPMount
}

// HTTPMount is how the plugin exposes its HTTP surface.
type HTTPMount struct {
	Handler *httpx.Router

	// Your default mount base. The end user can override this in Aegis config.
	// Example: "/magic" => routes live under /auth/magic/* by default.
	DefaultBase string
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
	SignUpWithEmailAndPassword(ctx context.Context, user *schemas.User, additionalFields map[string]any) (*AuthenticationResult, error)

	// HashPassword hashes the given password and returns the hash.
	// This is used to hash the password before storing it in the database.
	//
	// You only call this if you are not using the SignUpWithEmailAndPassword method.
	HashPassword(password string) (string, error)

	// ComparePassword compares the given password with the given hash and returns true if they match, false otherwise.
	ComparePassword(password string, hash string) (bool, error)

	// RequestPasswordReset requests a password reset for the given email.
	// Returns the verification object if the request is successful.
	RequestPasswordReset(ctx context.Context, email string) (*schemas.Verification, error)

	// ResetPassword resets the password using the given token and new password.
	ResetPassword(ctx context.Context, token string, newPassword string) error

	// UpdatePassword updates the password for the given user.
	UpdatePassword(ctx context.Context, user *schemas.User, currentPassword string, newPassword string) error

	// RequestEmailVerification requests an email verification for the given user.
	RequestEmailVerification(ctx context.Context, user *schemas.User) (*schemas.Verification, error)

	// VerifyEmail verifies the email using the given token.
	VerifyEmail(ctx context.Context, token string) error
}
