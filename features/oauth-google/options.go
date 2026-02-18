package oauthgoogle

// ConfigOption configures the Google OAuth feature.
type ConfigOption func(*config)

type config struct {
	clientID             string
	clientSecret         string
	redirectURL          string
	scopes               []string
	prompt               string
	accessType           string
	includeGrantedScopes bool
}

// WithClientID sets the Google OAuth2 client ID.
func WithClientID(id string) ConfigOption {
	return func(c *config) {
		c.clientID = id
	}
}

// WithClientSecret sets the Google OAuth2 client secret.
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

// WithPrompt sets the OAuth2 prompt (e.g. "consent", "select_account").
func WithPrompt(prompt Prompt) ConfigOption {
	return func(c *config) {
		c.prompt = string(prompt)
	}
}

// WithAccessType sets the OAuth2 access type (e.g. "offline", "online").
func WithAccessType(accessType AccessType) ConfigOption {
	return func(c *config) {
		c.accessType = string(accessType)
	}
}

// WithIncludeGrantedScopes sets the OAuth2 include granted scopes (e.g. "true", "false").
func WithIncludeGrantedScopes(includeGrantedScopes bool) ConfigOption {
	return func(c *config) {
		c.includeGrantedScopes = includeGrantedScopes
	}
}
