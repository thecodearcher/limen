package aegis

import (
	"context"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"time"
)

type rateLimiter struct {
	config          *RateLimiterConfig
	store           RateLimiterStore
	httpCore        *AegisHTTPCore
	rules           []*RateLimitRule
	disableForPaths []*regexp.Regexp
}

func newRateLimiter(config *RateLimiterConfig, httpCore *AegisHTTPCore, rules map[string]*RateLimitRule) *rateLimiter {
	sortedRules := slices.Collect(maps.Values(rules))
	sortRulesBySpecificity(sortedRules)

	return &rateLimiter{
		config:   config,
		httpCore: httpCore,
		store:    determineRateLimiterStore(config, httpCore.core),
		rules:    sortedRules,
	}
}

func determineRateLimiterStore(config *RateLimiterConfig, core *AegisCore) RateLimiterStore {
	if config.CustomStore != nil {
		return config.CustomStore
	}

	switch config.Store {
	case RateLimiterStoreTypeDatabase:
		return NewDatabaseRateLimiterStore(core)
	case RateLimiterStoreTypeMemory:
		fallthrough
	default:
		return NewMemoryRateLimiterStore()
	}
}

func (r *rateLimiter) Check(ctx context.Context, key string, rule *RateLimitRule) (time.Duration, error) {
	limit, err := r.store.Get(ctx, key)

	if err == ErrRateLimitNotFound {
		return r.createNewLimit(ctx, key)
	}

	if err != nil {
		return 0, err
	}

	remainingTime := r.computeRemainingTime(limit, rule.window)
	if remainingTime <= 0 {
		return r.resetAndIncrement(ctx, limit, rule.window)
	}

	if limit.Count >= rule.maxRequests {
		return remainingTime, ErrRateLimitExceeded
	}

	if err := r.incrementCounter(ctx, limit); err != nil {
		return 0, err
	}
	return remainingTime, nil
}

func (r *rateLimiter) createNewLimit(ctx context.Context, key string) (time.Duration, error) {
	limit := &RateLimit{
		Key:           key,
		Count:         1,
		LastRequestAt: time.Now().UnixMilli(),
	}

	if err := r.store.Create(ctx, limit); err != nil {
		return 0, err
	}

	return r.config.Window, nil
}

func (r *rateLimiter) resetAndIncrement(ctx context.Context, limit *RateLimit, window time.Duration) (time.Duration, error) {
	limit.ResetCounter()
	limit.Touch()

	if err := r.store.Update(ctx, limit.Key, limit); err != nil {
		return 0, err
	}

	return window, nil
}

func (r *rateLimiter) incrementCounter(ctx context.Context, limit *RateLimit) error {
	limit.Touch()
	return r.store.Update(ctx, limit.Key, limit)
}

func (r *rateLimiter) computeRemainingTime(limit *RateLimit, window time.Duration) time.Duration {
	remainingTime := window - time.Since(time.UnixMilli(limit.LastRequestAt))
	if remainingTime < 0 {
		return 0
	}
	return remainingTime
}

func (r *rateLimiter) findApplicableRule(req *http.Request) *RateLimitRule {
	for _, rule := range r.rules {
		if pathMatcher(req, rule.pathRegex) {
			if rule.limitProvider != nil {
				limit, window := rule.limitProvider(req)
				return NewRateLimitRule(rule.path, limit, window)
			}
			return rule
		}
	}

	return NewRateLimitRule("", r.config.MaxRequests, r.config.Window)
}

func (rl *rateLimiter) handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		rule := rl.findApplicableRule(r)
		if !rule.enabled {
			next.ServeHTTP(w, r)
			return
		}

		key := rl.config.KeyGenerator(r)
		if rule.path != "" && rule.path != "**" {
			key = key + "::" + rule.path
		}
		remainingTime, err := rl.Check(r.Context(), key, rule)
		if err != nil {
			w.Header().Set("Retry-After", strconv.Itoa(int(remainingTime.Seconds())))
			rl.httpCore.Responder.Error(w, r, NewAegisError("Too many requests. Please try again later.", http.StatusTooManyRequests, nil))
			return
		}

		next.ServeHTTP(w, r)
	})
}
