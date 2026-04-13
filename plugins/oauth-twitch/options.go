package oauthtwitch

// ConfigOption configures the Twitch OAuth plugin.
type ConfigOption func(*config)

type config struct {
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	options      map[string]string
}

// WithClientID sets the Twitch OAuth2 client ID.
// Defaults to env var TWITCH_CLIENT_ID.
func WithClientID(id string) ConfigOption {
	return func(c *config) {
		c.clientID = id
	}
}

// WithClientSecret sets the Twitch OAuth2 client secret.
// Defaults to env var TWITCH_CLIENT_SECRET.
func WithClientSecret(secret string) ConfigOption {
	return func(c *config) {
		c.clientSecret = secret
	}
}

// WithRedirectURL sets the callback URL registered in the Twitch Developer Console.
func WithRedirectURL(url string) ConfigOption {
	return func(c *config) {
		c.redirectURL = url
	}
}

// WithScopes sets the OAuth2 scopes.
// Default: "openid", "user:read:email".
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
