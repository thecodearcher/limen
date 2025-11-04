package aegis

import (
	"net/http"
	"time"
)

type SessionConfig struct {
	Strategy SessionStrategyType
	// Duration: the absolute duration of the session
	Duration time.Duration
	// RefreshInterval: the interval at which the session will be refreshed
	RefreshInterval time.Duration
	// IdleTimeout: the time after which the session will be considered expired if no activity is detected
	IdleTimeout time.Duration
	// ActivityCheckInterval: the interval at which the session last access time will be updated
	ActivityCheckInterval time.Duration
	// StoreType: the type of session store to use if no custom store is provided
	StoreType SessionStoreType
	// CustomStore: a custom session store to use instead of the default store
	CustomStore SessionStore
	// CookieOptions: the cookie options to use
	CookieOptions *CookieConfig
	// TrustedOrigins: list of allowed origins for cross-site credentialed requests (CORS + CSRF header).
	TrustedOrigins []string
	// TokenGenerator: the token generator to use
	TokenGenerator TokenGenerator
}

type CookieConfig struct {
	Name        string
	Path        string
	Secure      bool
	HTTPOnly    bool
	SameSite    http.SameSite
	Partitioned bool // optional: set true for browsers supporting CHIPS/partitioned cookies
	// CrossSubdomain: share cookies across subdomains while keeping SameSite=Lax.
	// Set Cookie.Domain to ".example.com" (your eTLD+1) when true.
	CrossSubdomain *CrossDomainConfig

	// CrossDomain: allow cookies to be sent from entirely different sites (requires SameSite=None; Secure=true).
	// When enabled, Aegis will force CSRF.Enabled=true and require CORS credentials + TrustedOrigins.
	CrossDomain bool
}

type CrossDomainConfig struct {
	Enabled bool
	Domain  string
}

func NewDefaultSessionConfig(opts ...SessionConfigOption) *SessionConfig {
	config := &SessionConfig{
		Strategy:              SessionStrategyServerSide,
		Duration:              1 * time.Hour,
		RefreshInterval:       0,
		IdleTimeout:           0,
		ActivityCheckInterval: 1 * time.Hour,
		CookieOptions: &CookieConfig{
			Name:        "aegis_session",
			Path:        "/",
			Secure:      true,
			HTTPOnly:    true,
			SameSite:    http.SameSiteLaxMode,
			Partitioned: false,
			CrossSubdomain: &CrossDomainConfig{
				Enabled: false,
			},
			CrossDomain: false,
		},
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

func (c *SessionConfig) validate() error {
	return nil
}

type SessionConfigOption func(*SessionConfig)

func WithSessionStrategy(strategy SessionStrategyType) SessionConfigOption {
	return func(c *SessionConfig) {
		c.Strategy = strategy
	}
}

func WithCustomSessionStore(store SessionStore) SessionConfigOption {
	return func(c *SessionConfig) {
		c.CustomStore = store
	}
}

func WithSessionStoreType(storeType SessionStoreType) SessionConfigOption {
	return func(c *SessionConfig) {
		c.StoreType = storeType
	}
}

func WithSessionDuration(duration time.Duration) SessionConfigOption {
	return func(c *SessionConfig) {
		c.Duration = duration
	}
}

func WithSessionRefreshInterval(refreshInterval time.Duration) SessionConfigOption {
	return func(c *SessionConfig) {
		c.RefreshInterval = refreshInterval
	}
}

func WithSessionIdleTimeout(idleTimeout time.Duration) SessionConfigOption {
	return func(c *SessionConfig) {
		c.IdleTimeout = idleTimeout
	}
}

func WithSessionCookieName(cookieName string) SessionConfigOption {
	return func(c *SessionConfig) {
		c.CookieOptions.Name = cookieName
	}
}

func WithSessionCookieOptions(cookieOptions *CookieConfig) SessionConfigOption {
	return func(c *SessionConfig) {
		c.CookieOptions = cookieOptions
	}
}

func WithSessionCookieCrossSubdomainEnabled(subdomain string) SessionConfigOption {
	return func(c *SessionConfig) {
		c.CookieOptions.CrossSubdomain.Enabled = true
		c.CookieOptions.CrossSubdomain.Domain = subdomain
	}
}

func WithSessionCookieCrossDomainEnabled() SessionConfigOption {
	return func(c *SessionConfig) {
		c.CookieOptions.CrossDomain = true
		c.CookieOptions.SameSite = http.SameSiteNoneMode
		c.CookieOptions.Secure = true
		c.CookieOptions.Partitioned = true
	}
}

func WithSessionCookiePath(cookiePath string) SessionConfigOption {
	return func(c *SessionConfig) {
		c.CookieOptions.Path = cookiePath
	}
}

func WithSessionCookieSecure(cookieSecure bool) SessionConfigOption {
	return func(c *SessionConfig) {
		c.CookieOptions.Secure = cookieSecure
	}
}

func WithSessionCookieHTTPOnly(cookieHTTPOnly bool) SessionConfigOption {
	return func(c *SessionConfig) {
		c.CookieOptions.HTTPOnly = cookieHTTPOnly
	}
}

func WithSessionCookieSameSite(cookieSameSite http.SameSite) SessionConfigOption {
	return func(c *SessionConfig) {
		c.CookieOptions.SameSite = cookieSameSite
	}
}
