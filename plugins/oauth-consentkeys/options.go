package oauthconsentkeys

// ConfigOption configures the ConsentKeys OAuth plugin.
type ConfigOption func(*config)

type config struct {
	clientID         string
	clientSecret     string
	authorizationURL string
	tokenURL         string
	userInfoURL      string
	redirectURL      string
	scopes           []string
	options          map[string]string
}

// WithClientID sets the ConsentKeys OAuth2 client ID.
// Defaults to env var CONSENTKEYS_CLIENT_ID.
func WithClientID(id string) ConfigOption {
	return func(c *config) {
		c.clientID = id
	}
}

// WithClientSecret sets the ConsentKeys OAuth2 client secret.
// Defaults to env var CONSENTKEYS_CLIENT_SECRET.
func WithClientSecret(secret string) ConfigOption {
	return func(c *config) {
		c.clientSecret = secret
	}
}

// WithAuthorizationURL overrides the authorization endpoint discovered from OIDC metadata.
func WithAuthorizationURL(url string) ConfigOption {
	return func(c *config) {
		c.authorizationURL = url
	}
}

// WithTokenURL overrides the token endpoint discovered from OIDC metadata.
func WithTokenURL(url string) ConfigOption {
	return func(c *config) {
		c.tokenURL = url
	}
}

// WithUserInfoURL overrides the userinfo endpoint discovered from OIDC metadata.
func WithUserInfoURL(url string) ConfigOption {
	return func(c *config) {
		c.userInfoURL = url
	}
}

// WithRedirectURL sets the callback URL registered in ConsentKeys.
func WithRedirectURL(url string) ConfigOption {
	return func(c *config) {
		c.redirectURL = url
	}
}

// WithScopes sets the OAuth2 scopes (defaults to "openid", "profile", "email").
func WithScopes(scopes ...string) ConfigOption {
	return func(c *config) {
		c.scopes = scopes
	}
}

// WithOption sets any additional OAuth2 authorization parameters.
func WithOption(key, value string) ConfigOption {
	return func(c *config) {
		if c.options == nil {
			c.options = make(map[string]string)
		}
		c.options[key] = value
	}
}
