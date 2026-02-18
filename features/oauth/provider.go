package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

// Provider is implemented by each OAuth provider plugin (Google, GitHub, etc.).
// The base module uses OAuth2Config for standard authorization URL and token exchange;
// GetUserInfo is the only provider-specific call.
type Provider interface {
	// Name returns the provider identifier (e.g., "google", "github").
	Name() string
	// OAuth2Config returns the OAuth2 client config and optional auth-code options
	// (e.g. AccessTypeOffline for Google). The base module uses these for
	// BuildAuthCodeURL and ExchangeCode.
	OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption)
	// GetUserInfo fetches the user's profile from the provider using the access token.
	GetUserInfo(ctx context.Context, token *TokenResponse) (*ProviderUserInfo, error)
}
