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

// AuthorizationURLBuilder is optional. If a Provider also implements it,
// the base module uses it to build the authorization URL instead of BuildAuthCodeURL.
type AuthorizationURLBuilder interface {
	BuildAuthorizationURL(ctx context.Context, state, codeVerifier, callbackRedirectURI string) (string, error)
}

// TokenExchanger is optional. If a Provider also implements TokenExchanger,
// the base module uses it for code-for-token exchange instead of ExchangeCode.
type TokenExchanger interface {
	ExchangeAuthorizationCode(ctx context.Context, code, codeVerifier, redirectURI string) (*TokenResponse, error)
}

// TokenRefresher is optional. If a Provider also implements TokenRefresher,
// the base module uses it to refresh an access token instead of the standard
// oauth2 refresh flow.
type TokenRefresher interface {
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
}

// PKCEEnabledProvider is optional. If a Provider implements it and PKCEEnabled() returns false,
// the authorization URL is built without code_challenge and the token exchange is
// performed without code_verifier (for providers like LinkedIn that do not support PKCE).
type PKCEEnabledProvider interface {
	PKCEEnabled() bool
}
