package oauthgoogle

// ConfigOption configures the Google OAuth plugin.
type ConfigOption func(*config)

type config struct {
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	options      map[string]string
}

// WithClientID sets the Google OAuth2 client ID.
// Defaults to env var GOOGLE_CLIENT_ID.
func WithClientID(id string) ConfigOption {
	return func(c *config) {
		c.clientID = id
	}
}

// WithClientSecret sets the Google OAuth2 client secret.
// Defaults to env var GOOGLE_CLIENT_SECRET.
func WithClientSecret(secret string) ConfigOption {
	return func(c *config) {
		c.clientSecret = secret
	}
}

// WithRedirectURL sets the callback URL registered in Google Cloud Console.
func WithRedirectURL(url string) ConfigOption {
	return func(c *config) {
		c.redirectURL = url
	}
}

// WithScopes sets the OAuth2 scopes (e.g. "openid", "email", "profile").
func WithScopes(scopes ...string) ConfigOption {
	return func(c *config) {
		c.scopes = scopes
	}
}

// WithOption sets any additional OAuth2 options such as "prompt", "access_type", etc.
func WithOption(key, value string) ConfigOption {
	return func(c *config) {
		if c.options == nil {
			c.options = make(map[string]string)
		}
		c.options[key] = value
	}
}
