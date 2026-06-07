package passkey

import (
	"context"
	"io"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/thecodearcher/limen"
)

// API is the public interface for the passkey plugin. Obtain a
// type-safe reference via Use().
type API interface {
	BeginRegistration(ctx context.Context, r *http.Request, w http.ResponseWriter, nameHint string) (*protocol.CredentialCreation, error)
	FinishRegistration(ctx context.Context, r *http.Request, body io.Reader) (*Passkey, error)
	BeginAuthentication(ctx context.Context, w http.ResponseWriter) (*protocol.CredentialAssertion, error)
	FinishAuthentication(ctx context.Context, r *http.Request, body io.Reader) (*limen.AuthenticationResult, error)
	ListPasskeys(ctx context.Context, userID any) ([]Passkey, error)
	DeletePasskey(ctx context.Context, userID, passkeyID any) error
	UpdatePasskey(ctx context.Context, userID, passkeyID any, newName string) (*Passkey, error)
}

// Use returns the passkey plugin's API from a Limen instance. Panics
// if the plugin was not registered in Config.Plugins.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginPasskey)
}
