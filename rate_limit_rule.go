package aegis

import (
	"net/http"
	"regexp"
	"time"
)

type LimitProvider func(req *http.Request) (maxRequests int, window time.Duration)

type RateLimitRule struct {
	enabled       bool
	path          string
	maxRequests   int
	window        time.Duration
	pathRegex     *regexp.Regexp
	limitProvider LimitProvider
}

func NewRateLimitRule(path string, maxRequests int, window time.Duration) *RateLimitRule {
	return &RateLimitRule{
		enabled:     true,
		path:        path,
		maxRequests: maxRequests,
		window:      window,
	}
}

func NewRateLimitRuleWithLimitProvider(path string, limitProvider LimitProvider) *RateLimitRule {
	return &RateLimitRule{
		enabled:       true,
		path:          path,
		limitProvider: limitProvider,
	}
}

func NewRateLimitRuleDisabledForPath(path string) *RateLimitRule {
	return &RateLimitRule{
		path:    path,
		enabled: false,
	}
}
