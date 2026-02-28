package credentialpassword

import (
	"context"

	"github.com/thecodearcher/aegis"
)

// API is the public interface for the credential-password plugin.
// Call the Use() function to obtain a type-safe reference from an Aegis instance.
type API interface {
	SignInWithCredentialAndPassword(ctx context.Context, credential string, password string) (*aegis.AuthenticationResult, error)

	SignUpWithCredentialAndPassword(ctx context.Context, user *aegis.User, additionalFields map[string]any) (*aegis.AuthenticationResult, error)

	HashPassword(password string) (string, error)

	ComparePassword(password string, hash *string) (bool, error)

	RequestPasswordReset(ctx context.Context, email string) (*aegis.Verification, error)

	ResetPassword(ctx context.Context, token string, newPassword string) error

	UpdatePassword(ctx context.Context, user *aegis.User, currentPassword string, newPassword string, revokeOtherSessions bool) error

	// SetPassword sets a password for a user who doesn't have one (e.g., signed up via OAuth).
	SetPassword(ctx context.Context, user *aegis.User, newPassword string, revokeOtherSessions bool) error

	RequestEmailVerification(ctx context.Context, user *aegis.User, shouldSendEmail bool) (*aegis.Verification, error)

	VerifyEmail(ctx context.Context, token string) error

	FindUserByUsername(ctx context.Context, username string) (*aegis.User, error)

	CheckUsernameAvailability(ctx context.Context, username string) (bool, error)
}

// Use returns a type-safe API for the credential-password plugin.
// Panics if the plugin was not registered in Config.Plugins,
// making it suitable for method chaining.
func Use(a *aegis.Aegis) API {
	return aegis.Use[API](a, aegis.PluginCredentialPassword)
}
