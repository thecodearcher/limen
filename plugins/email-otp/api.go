package emailotp

import (
	"context"

	"github.com/thecodearcher/limen"
)

// API is the public interface for the email-otp plugin. Obtain a type-safe
// reference via Use().
type API interface {
	SendOTP(ctx context.Context, email string, opts ...*SendOTPOptions) (*EmailOTPMessage, error)
	SignInWithOTP(ctx context.Context, email, otp string, opts ...*SignInOptions) (*limen.AuthenticationResult, error)
	VerifyOTP(ctx context.Context, email, otp string) (*limen.AuthenticationResult, error)
}

// Use returns the email-otp plugin's API from a Limen instance. Panics if the
// plugin was not registered in Config.Plugins.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginEmailOTP)
}
