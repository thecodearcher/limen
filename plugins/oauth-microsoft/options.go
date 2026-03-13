package oauthmicrosoft

// ConfigOption configures the Microsoft OAuth plugin.
type ConfigOption func(*config)

type config struct {
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	tenant       string
	authorityURL string
	options      map[string]string
}

// WithClientID sets the Microsoft OAuth2 client ID (Application ID).
func WithClientID(id string) ConfigOption {
	return func(c *config) {
		c.clientID = id
	}
}

// WithClientSecret sets the Microsoft OAuth2 client secret.
func WithClientSecret(secret string) ConfigOption {
	return func(c *config) {
		c.clientSecret = secret
	}
}

// WithRedirectURL sets the callback URL registered in the Azure Portal.
func WithRedirectURL(url string) ConfigOption {
	return func(c *config) {
		c.redirectURL = url
	}
}

// WithScopes sets the OAuth2 scopes (e.g. "openid", "profile", "email").
func WithScopes(scopes ...string) ConfigOption {
	return func(c *config) {
		c.scopes = scopes
	}
}

// WithTenant sets the Azure AD tenant for the authorization and token endpoints.
// Common values: "common" (default, all account types), "organizations" (work/school only),
// "consumers" (personal Microsoft accounts only), or a specific tenant GUID/domain.
func WithTenant(tenant string) ConfigOption {
	return func(c *config) {
		c.tenant = tenant
	}
}

// WithAuthorityURL sets a custom authority base URL for the authorization and token
// endpoints. Use this for Microsoft Entra External ID (CIAM) or other non-standard
// deployments.
//
// Example CIAM: "https://mytenant.ciamlogin.com/mytenant.onmicrosoft.com"
// Example standard: "https://login.microsoftonline.com/contoso.onmicrosoft.com"
//
// When set, this takes precedence over WithTenant.
func WithAuthorityURL(url string) ConfigOption {
	return func(c *config) {
		c.authorityURL = url
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
