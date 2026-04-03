package twofactor

import (
	"context"
	"net/http"

	"github.com/thecodearcher/limen"
)

// API is the public interface for the two-factor authentication plugin.
// Use the Use function to obtain a type-safe reference from a Limen instance.
type API interface {
	InitiateTwoFactorSetup(ctx context.Context, user *UserWithTwoFactor, password string) (*TwoFactorSetupURI, error)

	FinalizeTwoFactorSetup(ctx context.Context, user *UserWithTwoFactor, code string) error

	DisableTwoFactor(ctx context.Context, userID any, password string) error

	VerifyLoginWithTwoFactor(r *http.Request, w http.ResponseWriter, code string, method TwoFactorMethod) (*limen.AuthenticationResult, *limen.SessionResult, error)

	FindTwoFactorByUserID(ctx context.Context, userID any) (*TwoFactor, error)

	TOTP() *totp

	OTP() *otp

	BackupCodes() *backupCodes
}

// Use returns a type-safe API for the two-factor plugin.
// Panics if the plugin was not registered in Config.Plugins,
// making it suitable for method chaining.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginTwoFactor)
}
