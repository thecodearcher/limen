package oauthfacebook

// ConfigOption configures the Facebook OAuth plugin.
type ConfigOption func(*config)

type config struct {
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	options      map[string]string
}

// WithClientID sets the Facebook OAuth2 client ID (App ID).
// Defaults to env var FACEBOOK_CLIENT_ID.
func WithClientID(id string) ConfigOption {
	return func(c *config) {
		c.clientID = id
	}
}

// WithClientSecret sets the Facebook OAuth2 client secret (App Secret).
// Defaults to env var FACEBOOK_CLIENT_SECRET.
func WithClientSecret(secret string) ConfigOption {
	return func(c *config) {
		c.clientSecret = secret
	}
}

// WithRedirectURL sets the callback URL registered in Meta for Developers.
func WithRedirectURL(url string) ConfigOption {
	return func(c *config) {
		c.redirectURL = url
	}
}

// WithScopes sets the OAuth2 scopes (e.g. "email", "public_profile").
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
