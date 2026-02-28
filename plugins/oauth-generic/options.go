package oauthgeneric

import (
	"context"

	"github.com/thecodearcher/aegis/plugins/oauth"
)

// ConfigOption configures the generic OAuth provider.
type ConfigOption func(*config)

type config struct {
	name                  string
	clientID              string
	clientSecret          string
	authorizationURL      string
	tokenURL              string
	userInfoURL           string
	discoveryURL          string
	scopes                []string
	redirectURL           string
	options               map[string]string
	mapUserInfo           func(raw map[string]any) (*oauth.ProviderUserInfo, error)
	getUserInfo           func(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error)
	buildAuthorizationURL func(ctx context.Context, state, codeVerifier, callbackRedirectURI string) (string, error)
	exchangeTokens        func(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth.TokenResponse, error)
	refreshTokens         func(ctx context.Context, refreshToken string) (*oauth.TokenResponse, error)
}

func (c *config) resolveDiscovery() {
	if c.discoveryURL == "" {
		return
	}
	doc, err := fetchDiscoveryDocument(c.discoveryURL)
	if err != nil {
		panic("oauth-generic: discovery fetch failed: " + err.Error())
	}
	if c.authorizationURL == "" {
		c.authorizationURL = doc.AuthorizationEndpoint
	}
	if c.tokenURL == "" {
		c.tokenURL = doc.TokenEndpoint
	}
	if c.userInfoURL == "" {
		c.userInfoURL = doc.UserinfoEndpoint
	}
}

func (c *config) validate() {
	required := map[string]struct {
		value string
		hint  string
	}{
		"name":          {c.name, "WithName"},
		"client ID":     {c.clientID, "WithClientID"},
		"client secret": {c.clientSecret, "WithClientSecret"},
	}
	for field, r := range required {
		if r.value == "" {
			panic("oauth-generic: " + field + " is required (use " + r.hint + ")")
		}
	}
	if c.authorizationURL == "" && c.buildAuthorizationURL == nil {
		panic("oauth-generic: authorization URL is required (use WithAuthorizationURL or WithBuildAuthorizationURL)")
	}
	if c.tokenURL == "" && c.exchangeTokens == nil {
		panic("oauth-generic: token URL is required (use WithTokenURL or WithExchangeTokens)")
	}
	if c.getUserInfo == nil && c.userInfoURL == "" && c.mapUserInfo == nil {
		panic("oauth-generic: one of WithGetUserInfo, WithUserInfoURL + WithMapUserInfo, or WithMapUserInfo (for id_token) is required")
	}
	if c.getUserInfo == nil && c.userInfoURL != "" && c.mapUserInfo == nil {
		panic("oauth-generic: WithMapUserInfo is required when using WithUserInfoURL")
	}
}

func (c *config) resolveDefaults() {
	if len(c.scopes) == 0 {
		c.scopes = []string{"openid", "email", "profile"}
	}
}

// WithName sets the provider identifier (e.g. "discord", "slack").
func WithName(name string) ConfigOption {
	return func(c *config) {
		c.name = name
	}
}

// WithClientID sets the OAuth2 client ID.
func WithClientID(id string) ConfigOption {
	return func(c *config) {
		c.clientID = id
	}
}

// WithClientSecret sets the OAuth2 client secret.
func WithClientSecret(secret string) ConfigOption {
	return func(c *config) {
		c.clientSecret = secret
	}
}

// WithAuthorizationURL sets the provider's authorization endpoint.
func WithAuthorizationURL(url string) ConfigOption {
	return func(c *config) {
		c.authorizationURL = url
	}
}

// WithTokenURL sets the provider's token endpoint.
func WithTokenURL(url string) ConfigOption {
	return func(c *config) {
		c.tokenURL = url
	}
}

// WithUserInfoURL sets the endpoint to fetch user profile (GET with Bearer token).
func WithUserInfoURL(url string) ConfigOption {
	return func(c *config) {
		c.userInfoURL = url
	}
}

// WithDiscoveryURL sets the OpenID Connect discovery URL. When set, the provider fetches
// the discovery document and populates authorization, token, and userinfo endpoints
// from it (only for fields not already set explicitly).
func WithDiscoveryURL(url string) ConfigOption {
	return func(c *config) {
		c.discoveryURL = url
	}
}

// WithScopes sets the OAuth2 scopes (defaults to openid, email, profile if empty).
func WithScopes(scopes ...string) ConfigOption {
	return func(c *config) {
		c.scopes = scopes
	}
}

// WithRedirectURL sets the callback URL (auto-constructed by base OAuth module if omitted).
func WithRedirectURL(url string) ConfigOption {
	return func(c *config) {
		c.redirectURL = url
	}
}

// WithOption sets an additional auth URL parameter (e.g. "prompt", "access_type").
func WithOption(key, value string) ConfigOption {
	return func(c *config) {
		if c.options == nil {
			c.options = make(map[string]string)
		}
		c.options[key] = value
	}
}

// WithMapUserInfo sets a function that maps the provider's raw JSON userinfo response
// to a *oauth.ProviderUserInfo. Required when using WithUserInfoURL, unless
// WithGetUserInfo is set (which handles the full fetch + mapping itself).
func WithMapUserInfo(fn func(raw map[string]any) (*oauth.ProviderUserInfo, error)) ConfigOption {
	return func(c *config) {
		c.mapUserInfo = fn
	}
}

// WithGetUserInfo sets a custom function to fetch user info; overrides UserInfoURL when set.
func WithGetUserInfo(fn func(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error)) ConfigOption {
	return func(c *config) {
		c.getUserInfo = fn
	}
}

// WithBuildAuthorizationURL sets a custom function to build the authorization URL.
// When set, the base OAuth module will call this instead of BuildAuthCodeURL.
func WithBuildAuthorizationURL(fn func(ctx context.Context, state, codeVerifier, callbackRedirectURI string) (string, error)) ConfigOption {
	return func(c *config) {
		c.buildAuthorizationURL = fn
	}
}

// WithExchangeTokens sets a custom function to exchange the authorization code for tokens.
// When set, the base OAuth module will call this instead of the standard exchange.
func WithExchangeTokens(fn func(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth.TokenResponse, error)) ConfigOption {
	return func(c *config) {
		c.exchangeTokens = fn
	}
}

// WithRefreshTokens sets a custom function to refresh an access token using a refresh token.
// When set, the base OAuth module will call this instead of the standard oauth2 token refresh.
// Use this when the provider's refresh endpoint expects a different format or parameters.
func WithRefreshTokens(fn func(ctx context.Context, refreshToken string) (*oauth.TokenResponse, error)) ConfigOption {
	return func(c *config) {
		c.refreshTokens = fn
	}
}
