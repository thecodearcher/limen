package aegis

import (
	"log"
	"time"
)

type RateLimiterConfig struct {
	// Enabled: whether the rate limiter is enabled
	Enabled bool
	// MaxRequests: the maximum number of requests allowed within the window
	MaxRequests int
	// Window: the duration of the window
	Window time.Duration
	// Store: the type of store to use
	Store StoreType
	// CustomStore: a custom store to use
	CustomStore RateLimiterStore
	// KeyGenerator: a function to generate the key for the rate limiter
	KeyGenerator RequestExtractorFn
	// CustomRules: a map of custom rules to use for specific routes/routeIDs
	customRules map[string]*RateLimitRule
}

type RateLimiterOption func(*RateLimiterConfig)

func NewDefaultRateLimiterConfig(opts ...RateLimiterOption) *RateLimiterConfig {
	config := &RateLimiterConfig{
		Enabled:      true,
		MaxRequests:  100,
		Window:       time.Minute,
		KeyGenerator: ipExtractorFromRemoteAddr,
		Store:        StoreTypeCache,
		customRules:  make(map[string]*RateLimitRule),
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

func (c *RateLimiterConfig) validate() {
	if !c.Enabled {
		return
	}
	if c.MaxRequests <= 0 {
		log.Panicf("max requests must be greater than zero")
	}

	if c.Window <= 0 {
		log.Panicf("window duration must be greater than zero")
	}

	if c.KeyGenerator == nil {
		log.Panicf("key generator function is required")
	}
}

// WithRateLimiterEnabled sets whether the rate limiter is enabled
func WithRateLimiterEnabled(enabled bool) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		c.Enabled = enabled
	}
}

// WithRateLimiterMaxRequests sets the maximum number of requests allowed within the window
// default is 100
func WithRateLimiterMaxRequests(maxRequests int) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		c.MaxRequests = maxRequests
	}
}

// WithRateLimiterWindow sets the duration of the window
// default is 1 minute
func WithRateLimiterWindow(window time.Duration) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		c.Window = window
	}
}

// WithRateLimiterStore sets the type of store to use.
// Default is StoreTypeCache.
func WithRateLimiterStore(store StoreType) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		c.Store = store
	}
}

// WithRateLimiterCustomStore sets a custom store to use
func WithRateLimiterCustomStore(store RateLimiterStore) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		c.CustomStore = store
	}
}

// WithRateLimiterCustomRule sets a custom rule to use for a specific path
func WithRateLimiterCustomRule(path string, maxRequests int, window time.Duration) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		c.customRules[path] = NewRateLimitRule(path, maxRequests, window)
	}
}

// WithRateLimiterCustomRuleWithLimitProvider sets a custom rule to use for a specific path with a limit provider
// this is useful when you need to dynamically determine the limit and window based on the request
func WithRateLimiterCustomRuleWithLimitProvider(path string, limitProvider LimitProvider) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		c.customRules[path] = NewRateLimitRuleWithLimitProvider(path, limitProvider)
	}
}

// WithRateLimiterDisableForPaths disables the rate limiter for specific paths
func WithRateLimiterDisableForPaths(paths ...string) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		for _, path := range paths {
			c.customRules[path] = NewRateLimitRuleDisabledForPath(path)
		}
	}
}

// WithRateLimiterKeyGenerator sets the function to generate the key for the rate limiter
func WithRateLimiterKeyGenerator(keyGenerator RequestExtractorFn) RateLimiterOption {
	return func(c *RateLimiterConfig) {
		c.KeyGenerator = keyGenerator
	}
}
