package aegis

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

type sessionConfig struct {
	Strategy SessionStrategyType
	// Duration: the absolute duration of the session
	Duration time.Duration
	// TemporaryAuthDuration: the duration of the temporary auth session i.e: for two-factor authentication, email verification etc.
	// If not set, the duration will be set to 5 minutes
	TemporaryAuthDuration time.Duration
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
	// IPAddressExtractor: the function to extract the IP address from the request
	IPAddressExtractor func(request *http.Request) string
	// UserAgentExtractor: the function to extract the user agent from the request
	UserAgentExtractor func(request *http.Request) string
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
	// When enabled, Aegis will require TrustedOrigins.
	CrossDomain bool
}

type CrossDomainConfig struct {
	Enabled bool
	Domain  string
}

func NewDefaultSessionConfig(opts ...SessionConfigOption) *sessionConfig {
	config := &sessionConfig{
		Strategy:              SessionStrategyServerSide,
		Duration:              1 * time.Hour,
		TemporaryAuthDuration: 5 * time.Minute,
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
		IPAddressExtractor: func(request *http.Request) string {
			if ip := request.Header.Get("X-Forwarded-For"); ip != "" {
				return ip
			}
			if ip := request.Header.Get("X-Real-IP"); ip != "" {
				return ip
			}
			ip, _, _ := net.SplitHostPort(request.RemoteAddr)
			return ip
		},
		UserAgentExtractor: func(request *http.Request) string {
			return request.UserAgent()
		},
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

func (c *sessionConfig) validate() error {
	if c.CookieOptions.CrossDomain && len(c.TrustedOrigins) == 0 {
		return fmt.Errorf("trusted origins are required when cross domain is enabled")
	}

	if c.RefreshInterval > c.Duration {
		return fmt.Errorf("refresh interval cannot be greater than duration")
	}

	if c.IdleTimeout > c.Duration {
		return fmt.Errorf("idle timeout cannot be greater than duration")
	}

	if c.ActivityCheckInterval > c.Duration {
		return fmt.Errorf("activity check interval cannot be greater than duration")
	}

	return nil
}

type SessionConfigOption func(*sessionConfig)

func WithSessionStrategy(strategy SessionStrategyType) SessionConfigOption {
	return func(c *sessionConfig) {
		c.Strategy = strategy
	}
}

func WithCustomSessionStore(store SessionStore) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CustomStore = store
	}
}

func WithSessionStoreType(storeType SessionStoreType) SessionConfigOption {
	return func(c *sessionConfig) {
		c.StoreType = storeType
	}
}

func WithSessionDuration(duration time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.Duration = duration
	}
}

func WithSessionRefreshInterval(refreshInterval time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.RefreshInterval = refreshInterval
	}
}

func WithSessionIdleTimeout(idleTimeout time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.IdleTimeout = idleTimeout
	}
}

func WithSessionCookieName(cookieName string) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CookieOptions.Name = cookieName
	}
}

func WithSessionCookieOptions(cookieOptions *CookieConfig) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CookieOptions = cookieOptions
	}
}

func WithSessionCookieCrossSubdomainEnabled(subdomain string) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CookieOptions.CrossSubdomain.Enabled = true
		c.CookieOptions.CrossSubdomain.Domain = subdomain
	}
}

func WithSessionCookieCrossDomainEnabled() SessionConfigOption {
	return func(c *sessionConfig) {
		c.CookieOptions.CrossDomain = true
		c.CookieOptions.SameSite = http.SameSiteNoneMode
		c.CookieOptions.Secure = true
		c.CookieOptions.Partitioned = true
	}
}

func WithSessionTrustedOrigins(origins []string) SessionConfigOption {
	return func(c *sessionConfig) {
		c.TrustedOrigins = origins
	}
}

func WithSessionCookiePath(cookiePath string) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CookieOptions.Path = cookiePath
	}
}

func WithSessionCookieSecure(cookieSecure bool) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CookieOptions.Secure = cookieSecure
	}
}

func WithSessionCookieHTTPOnly(cookieHTTPOnly bool) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CookieOptions.HTTPOnly = cookieHTTPOnly
	}
}

func WithSessionCookieSameSite(cookieSameSite http.SameSite) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CookieOptions.SameSite = cookieSameSite
	}
}

func WithSessionIPAddressExtractor(ipAddressExtractor func(request *http.Request) string) SessionConfigOption {
	return func(c *sessionConfig) {
		c.IPAddressExtractor = ipAddressExtractor
	}
}

func WithSessionUserAgentExtractor(userAgentExtractor func(request *http.Request) string) SessionConfigOption {
	return func(c *sessionConfig) {
		c.UserAgentExtractor = userAgentExtractor
	}
}

func WithSessionTemporaryAuthDuration(temporaryAuthDuration time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.TemporaryAuthDuration = temporaryAuthDuration
	}
}

func WithSessionActivityCheckInterval(activityCheckInterval time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.ActivityCheckInterval = activityCheckInterval
	}
}
