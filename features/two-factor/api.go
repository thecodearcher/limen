package twofactor

import (
	"context"
	"net/http"

	"github.com/thecodearcher/aegis"
)

// API is the public interface for the two-factor authentication feature.
// Use the Use function to obtain a type-safe reference from an Aegis instance.
type API interface {
	InitiateTwoFactorSetup(ctx context.Context, user *UserWithTwoFactor, password string) (*TwoFactorSetupURI, error)

	FinalizeTwoFactorSetup(ctx context.Context, user *UserWithTwoFactor, code string) error

	DisableTwoFactor(ctx context.Context, userID any, password string) error

	VerifyLoginWithTwoFactor(r *http.Request, w http.ResponseWriter, code string, method TwoFactorMethod) (*aegis.AuthenticationResult, *aegis.SessionResult, error)

	FindTwoFactorByUserID(ctx context.Context, userID any) (*TwoFactor, error)
}

// Use returns a type-safe API for the two-factor feature.
// Panics if the feature was not registered in Config.Features,
// making it suitable for method chaining.
func Use(a *aegis.Aegis) API {
	return aegis.Use[API](a, aegis.FeatureTwoFactor)
}
