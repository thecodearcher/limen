package limen

import (
	"fmt"
	"net/http"
	"time"
)

type RequestExtractorFn func(request *http.Request) string

type sessionConfig struct {
	// Duration: the absolute duration of the session
	Duration time.Duration
	// UpdateAge: the time before expiration when session should be extended on use
	UpdateAge time.Duration
	// IdleTimeout: the time after which the session will be considered expired if no activity is detected
	IdleTimeout time.Duration
	// ActivityCheckInterval: the interval at which the session last access time will be updated
	ActivityCheckInterval time.Duration
	// StoreType: the type of session store to use if no custom store is provided
	StoreType StoreType
	// CustomStore: a custom session store to use instead of the default store
	CustomStore SessionStore
	// IPAddressExtractor: the function to extract the IP address from the request
	IPAddressExtractor RequestExtractorFn
	// UserAgentExtractor: the function to extract the user agent from the request
	UserAgentExtractor RequestExtractorFn
	// ShortSessionDuration: when > 0, sign-in with remember_me=false uses this shorter TTL instead of Duration. The session is not extended. 0 = remember-me plugin disabled.
	ShortSessionDuration time.Duration
	// BearerEnabled: when true, the opaque session manager also accepts
	// Authorization: Bearer <token> on requests, and session responses
	// include the token in Set-Auth-Token / Set-Refresh-Token headers
	// alongside cookies. Use when the client or API does not support
	// cookies or requires Bearer token authentication.
	BearerEnabled bool
}

func NewDefaultSessionConfig(opts ...SessionConfigOption) *sessionConfig {
	config := &sessionConfig{
		Duration:              7 * 24 * time.Hour, // 7 days
		UpdateAge:             24 * time.Hour,     // 1 day
		IdleTimeout:           0,                  // no idle timeout
		ActivityCheckInterval: 0,                  // no activity check interval
		StoreType:             StoreTypeDatabase,
		IPAddressExtractor:    ipExtractorFromRemoteAddr,
		ShortSessionDuration:  24 * time.Hour,
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
	if c.UpdateAge > c.Duration {
		return fmt.Errorf("update age cannot be greater than duration")
	}

	if c.IdleTimeout > c.Duration {
		return fmt.Errorf("idle timeout cannot be greater than duration")
	}

	if c.ActivityCheckInterval > c.Duration {
		return fmt.Errorf("activity check interval cannot be greater than duration")
	}

	if c.IdleTimeout > 0 && c.ActivityCheckInterval > 0 && c.ActivityCheckInterval >= c.IdleTimeout {
		return fmt.Errorf("activity check interval must be less than idle timeout")
	}
	if c.IdleTimeout > 0 && c.UpdateAge > 0 && c.UpdateAge >= c.IdleTimeout {
		return fmt.Errorf("update age must be less than idle timeout")
	}

	if c.ShortSessionDuration > 0 && c.ShortSessionDuration >= c.Duration {
		return fmt.Errorf("short session duration must be less than session duration")
	}

	return nil
}

type SessionConfigOption func(*sessionConfig)

func WithCustomSessionStore(store SessionStore) SessionConfigOption {
	return func(c *sessionConfig) {
		c.CustomStore = store
	}
}

func WithSessionStoreType(storeType StoreType) SessionConfigOption {
	return func(c *sessionConfig) {
		c.StoreType = storeType
	}
}

func WithSessionDuration(duration time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.Duration = duration
	}
}

func WithSessionUpdateAge(updateAge time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.UpdateAge = updateAge
	}
}

func WithSessionIdleTimeout(idleTimeout time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.IdleTimeout = idleTimeout
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

func WithSessionActivityCheckInterval(activityCheckInterval time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.ActivityCheckInterval = activityCheckInterval
	}
}

// WithSessionShortDuration sets the short TTL for non-remembered sessions.
// Must be less than global session Duration. 0 = remember-me plugin disabled.
func WithSessionShortDuration(d time.Duration) SessionConfigOption {
	return func(c *sessionConfig) {
		c.ShortSessionDuration = d
	}
}

// WithBearerEnabled enables Bearer token support for opaque sessions.
// When enabled, the session manager accepts Authorization: Bearer <token>
// in addition to cookies, and session responses include the token in
// Set-Auth-Token / Set-Refresh-Token headers. Use when the client or API
// does not support cookies or requires Bearer token authentication.
func WithBearerEnabled() SessionConfigOption {
	return func(c *sessionConfig) {
		c.BearerEnabled = true
	}
}
