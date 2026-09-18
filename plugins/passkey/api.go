package passkey

import (
	"context"
	"net/http"

	"github.com/thecodearcher/limen"
)

// API is the public interface for the passkey plugin.
type API interface {
	BeginRegistration(r *http.Request, user *limen.User, request *RegisterPasskeyRequest) (*CredentialCreation, *PasskeyChallenge, error)
	BeginPublicRegistration(r *http.Request, request *RegisterPasskeyRequest) (*CredentialCreation, *PasskeyChallenge, error)
	FinishRegistration(r *http.Request, user *limen.User, request *FinishRegistrationRequest) (*Passkey, error)
	FinishPublicRegistration(r *http.Request, request *FinishRegistrationRequest) (*limen.User, *Passkey, error)

	BeginAuthentication(r *http.Request) (*CredentialAssertion, *PasskeyChallenge, error)
	FinishAuthentication(r *http.Request) (*limen.User, error)

	FindPasskeysByUserID(ctx context.Context, userID any) ([]*Passkey, error)
	FindPasskeyByCredentialID(ctx context.Context, credentialID string) (*Passkey, error)
	ListPasskeys(ctx context.Context, user *limen.User, opts *limen.QueryOptions) (*limen.Page[*Passkey], error)
	UpdatePasskey(ctx context.Context, user *limen.User, id any, request *UpdatePasskeyRequest) (*Passkey, error)
	DeletePasskey(ctx context.Context, user *limen.User, id any) error
}

// Use returns a type-safe API for the passkey plugin.
// Panics if the plugin was not registered in Config.Plugins,
// making it suitable for method chaining.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginPasskey)
}

var _ API = (*passkeyPlugin)(nil)
