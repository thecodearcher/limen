package aegis

import (
	"net/http"
	"time"
)

const (
	shortSessionCookieName = "aegis_short_session"
	shortSessionMaxAgeSec  = 1 * time.Hour // 1 hour
)

type HTTPConfigOption func(*httpConfig)

type httpConfig struct {
	// Global middleware
	middleware []Middleware
	// The base path where all the routes will be mounted
	basePath string
	// overrides for specific plugins
	overrides map[string]*PluginHTTPOverride
	// Paths to be disabled by their ID or pattern
	disabledPaths []string
	// Response envelope configuration
	responseEnvelope *responseEnvelopeConfig
	// SessionTransformer customizes the session response payload before it's sent to the client.
	// Returns a map[string]any for the response body, or an AegisError to handle an error condition.
	sessionTransformer SessionTransformer
	// HTTPHooks are functions that are called before and after the request is processed
	hooks *Hooks
	// RateLimiter configuration
	rateLimiter *RateLimiterConfig
	// trustedOrigins: list of trusted origins.
	trustedOrigins []string
	// CSRFProtection: enable CSRF protection.
	csrfProtection bool
	// OriginCheck: enable origin check.
	originCheck bool
	// CookieConfig: configuration for cookies
	cookieConfig *cookieConfig
}

type cookieConfig struct {
	sessionCookieName string
	path              string
	secure            bool
	httpOnly          bool
	sameSite          http.SameSite
	partitioned       bool // optional: set true for browsers supporting CHIPS/partitioned cookies
	// crossSubdomain: share cookies across subdomains while keeping SameSite=Lax.
	// Set Cookie.Domain to ".example.com" (your eTLD+1) when true.
	crossSubdomain *crossDomainConfig
	// crossDomain: allow cookies to be sent from entirely different sites (requires SameSite=None; Secure=true).
	// When enabled, Aegis will require TrustedOrigins.
	crossDomain bool
}

type crossDomainConfig struct {
	enabled bool
	domain  string
}

// SessionTransformer customizes the session response payload.
type SessionTransformer func(user map[string]any, sessionResult *SessionResult) (map[string]any, error)

type EnvelopeFields struct {
	Data    string
	Message string
}

type responseEnvelopeConfig struct {
	mode   EnvelopeMode
	fields EnvelopeFields
}

type PluginHTTPOverride struct {
	BasePath string
	// Middleware to be applied to the plugin's routes
	Middleware []Middleware
}

func NewDefaultHTTPConfig(opts ...HTTPConfigOption) *httpConfig {
	config := &httpConfig{
		middleware:    []Middleware{},
		basePath:      "/auth",
		overrides:     map[string]*PluginHTTPOverride{},
		disabledPaths: []string{},
		responseEnvelope: &responseEnvelopeConfig{
			mode: EnvelopeOff,
		},
		rateLimiter:    NewDefaultRateLimiterConfig(),
		trustedOrigins: []string{},
		csrfProtection: true,
		originCheck:    true,
		cookieConfig: &cookieConfig{
			sessionCookieName: "aegis_session",
			path:              "/",
			secure:            true,
			httpOnly:          true,
			sameSite:          http.SameSiteLaxMode,
			partitioned:       false,
			crossSubdomain: &crossDomainConfig{
				enabled: false,
			},
			crossDomain: false,
		},
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

func WithHTTPBasePath(basePath string) HTTPConfigOption {
	return func(c *httpConfig) {
		c.basePath = basePath
	}
}

func WithHTTPTrustedOrigins(trustedOrigins []string) HTTPConfigOption {
	return func(c *httpConfig) {
		c.trustedOrigins = trustedOrigins
	}
}

func WithHTTPMiddleware(globalMW ...Middleware) HTTPConfigOption {
	return func(c *httpConfig) {
		c.middleware = append(c.middleware, globalMW...)
	}
}

func WithHTTPOverrides(overrides map[string]*PluginHTTPOverride) HTTPConfigOption {
	return func(c *httpConfig) {
		c.overrides = overrides
	}
}

// WithHTTPDisabledPaths adds paths to be disabled by their ID or pattern
func WithHTTPDisabledPaths(disabledPaths []string) HTTPConfigOption {
	return func(c *httpConfig) {
		c.disabledPaths = disabledPaths
	}
}

func WithHTTPResponseEnvelopeMode(mode EnvelopeMode) HTTPConfigOption {
	return func(c *httpConfig) {
		c.responseEnvelope.mode = mode
	}
}

func WithHTTPResponseEnvelopeFields(fields EnvelopeFields) HTTPConfigOption {
	return func(c *httpConfig) {
		c.responseEnvelope.fields = fields
	}
}

func WithHTTPSessionTransformer(transformer SessionTransformer) HTTPConfigOption {
	return func(c *httpConfig) {
		c.sessionTransformer = transformer
	}
}

func WithHTTPHooks(hooks *Hooks) HTTPConfigOption {
	return func(c *httpConfig) {
		c.hooks = hooks
	}
}

func WithHTTPRateLimiter(opts ...RateLimiterOption) HTTPConfigOption {
	return func(c *httpConfig) {
		c.rateLimiter = NewDefaultRateLimiterConfig(opts...)
	}
}

func WithHTTPCSRFProtection(csrfProtection bool) HTTPConfigOption {
	return func(c *httpConfig) {
		c.csrfProtection = csrfProtection
	}
}

func WithHTTPOriginCheck(originCheck bool) HTTPConfigOption {
	return func(c *httpConfig) {
		c.originCheck = originCheck
	}
}

// WithHTTPSessionCookieName sets the name of the session cookie
func WithHTTPSessionCookieName(name string) HTTPConfigOption {
	return func(c *httpConfig) {
		c.cookieConfig.sessionCookieName = name
	}
}

func WithHTTPCookiePath(path string) HTTPConfigOption {
	return func(c *httpConfig) {
		c.cookieConfig.path = path
	}
}

func WithHTTPCookieSecure(secure bool) HTTPConfigOption {
	return func(c *httpConfig) {
		c.cookieConfig.secure = secure
	}
}

func WithHTTPCookieHTTPOnly(httpOnly bool) HTTPConfigOption {
	return func(c *httpConfig) {
		c.cookieConfig.httpOnly = httpOnly
	}
}

func WithHTTPCookieSameSite(sameSite http.SameSite) HTTPConfigOption {
	return func(c *httpConfig) {
		c.cookieConfig.sameSite = sameSite
	}
}

func WithHTTPCookiePartitioned(partitioned bool) HTTPConfigOption {
	return func(c *httpConfig) {
		c.cookieConfig.partitioned = partitioned
	}
}

func WithHTTPCookieCrossSubdomainEnabled(subdomain string) HTTPConfigOption {
	return func(c *httpConfig) {
		if c.cookieConfig.crossSubdomain == nil {
			c.cookieConfig.crossSubdomain = &crossDomainConfig{}
		}
		c.cookieConfig.crossSubdomain.enabled = true
		c.cookieConfig.crossSubdomain.domain = subdomain
	}
}

func WithHTTPCookieCrossDomainEnabled() HTTPConfigOption {
	return func(c *httpConfig) {
		c.cookieConfig.crossDomain = true
		c.cookieConfig.sameSite = http.SameSiteNoneMode
		c.cookieConfig.secure = true
		c.cookieConfig.partitioned = true
	}
}
