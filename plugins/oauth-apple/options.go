package oauthapple

// ConfigOption configures the Apple OAuth plugin.
type ConfigOption func(*config)

type config struct {
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	options      map[string]string
}

// WithClientID sets the Apple Services ID (the identifier for your app).
// Defaults to env var APPLE_CLIENT_ID.
func WithClientID(id string) ConfigOption {
	return func(c *config) {
		c.clientID = id
	}
}

// WithClientSecret sets the client secret JWT for Apple Sign In.
// Defaults to env var APPLE_CLIENT_SECRET.
func WithClientSecret(secret string) ConfigOption {
	return func(c *config) {
		c.clientSecret = secret
	}
}

// WithRedirectURL sets the callback URL registered in Apple Developer Console.
func WithRedirectURL(url string) ConfigOption {
	return func(c *config) {
		c.redirectURL = url
	}
}

// WithScopes sets the OAuth2 scopes. Apple supports "name" and "email".
func WithScopes(scopes ...string) ConfigOption {
	return func(c *config) {
		c.scopes = scopes
	}
}

// WithOption sets any additional authorization parameters.
func WithOption(key, value string) ConfigOption {
	return func(c *config) {
		if c.options == nil {
			c.options = make(map[string]string)
		}
		c.options[key] = value
	}
}
