package limen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRateLimiterStore(t *testing.T) RateLimiterStore {
	t.Helper()
	l := newTestLimen(t)
	return newRateLimiterCacheStore(l.core)
}

func newTestRateLimiter(t *testing.T, maxReqs int, window time.Duration, rules ...*RateLimitRule) *rateLimiter {
	t.Helper()
	return &rateLimiter{
		config: &RateLimiterConfig{
			Enabled:      true,
			MaxRequests:  maxReqs,
			Window:       window,
			KeyGenerator: ipExtractorFromRemoteAddr,
		},
		store: newTestRateLimiterStore(t),
		rules: rules,
	}
}

// ---------------------------------------------------------------------------
// Rate limiter core logic
// ---------------------------------------------------------------------------

func TestRateLimiter_Check_NewKey(t *testing.T) {
	t.Parallel()

	rl := newTestRateLimiter(t, 5, time.Minute)

	rule := NewRateLimitRule("", 5, time.Minute)
	remaining, err := rl.Check(context.Background(), "test-key", rule)

	require.NoError(t, err)
	assert.Equal(t, time.Minute, remaining)

	got, err := rl.store.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Equal(t, 1, got.Count)
}

func TestRateLimiter_Check_IncrementExisting(t *testing.T) {
	t.Parallel()

	rl := newTestRateLimiter(t, 5, time.Minute)
	rule := NewRateLimitRule("", 5, time.Minute)

	for range 3 {
		_, err := rl.Check(context.Background(), "inc-key", rule)
		require.NoError(t, err)
	}

	got, err := rl.store.Get(context.Background(), "inc-key")
	require.NoError(t, err)
	assert.Equal(t, 3, got.Count)
}

func TestRateLimiter_Check_ExceedLimit(t *testing.T) {
	t.Parallel()

	rl := newTestRateLimiter(t, 3, time.Minute)
	rule := NewRateLimitRule("", 3, time.Minute)

	for range 3 {
		_, err := rl.Check(context.Background(), "exceed-key", rule)
		require.NoError(t, err)
	}

	_, err := rl.Check(context.Background(), "exceed-key", rule)
	assert.ErrorIs(t, err, ErrRateLimitExceeded)
}

func TestRateLimiter_Check_WindowReset(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		rl := newTestRateLimiter(t, 3, 10*time.Millisecond)
		rule := NewRateLimitRule("", 3, 10*time.Millisecond)

		for range 3 {
			_, err := rl.Check(context.Background(), "reset-key", rule)
			require.NoError(t, err)
		}

		time.Sleep(10 * time.Millisecond)

		remaining, err := rl.Check(context.Background(), "reset-key", rule)
		require.NoError(t, err)
		assert.Equal(t, 10*time.Millisecond, remaining, "window should have reset")

		got, err := rl.store.Get(context.Background(), "reset-key")
		require.NoError(t, err)
		assert.Equal(t, 1, got.Count, "counter should have reset")
	})
}

// ---------------------------------------------------------------------------
// Rule matching
// ---------------------------------------------------------------------------

func TestRateLimiter_FindApplicableRule_MatchesSpecificRule(t *testing.T) {
	t.Parallel()

	signinRule := NewRateLimitRule("/auth/signin", 5, time.Minute)
	signinRule.pathRegex = regexp.MustCompile("^/auth/signin$")

	rl := newTestRateLimiter(t, 100, time.Minute, signinRule)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/signin", http.NoBody)
	rule := rl.findApplicableRule(req)
	assert.Equal(t, "/auth/signin", rule.path)
	assert.Equal(t, 5, rule.maxRequests)
}

func TestRateLimiter_FindApplicableRule_FallsBackToDefault(t *testing.T) {
	t.Parallel()

	rl := newTestRateLimiter(t, 100, time.Minute)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/anything", http.NoBody)
	rule := rl.findApplicableRule(req)
	assert.Equal(t, 100, rule.maxRequests)
	assert.Equal(t, time.Minute, rule.window)
}

func TestRateLimiter_FindApplicableRule_WithLimitProvider(t *testing.T) {
	t.Parallel()

	dynamicRule := &RateLimitRule{
		enabled:   true,
		path:      "/api/dynamic",
		pathRegex: regexp.MustCompile("^/api/dynamic$"),
		limitProvider: func(req *http.Request) (int, time.Duration) {
			return 20, 2 * time.Minute
		},
	}

	rl := newTestRateLimiter(t, 100, time.Minute, dynamicRule)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/dynamic", http.NoBody)
	rule := rl.findApplicableRule(req)
	assert.Equal(t, 20, rule.maxRequests)
	assert.Equal(t, 2*time.Minute, rule.window)
}

// ---------------------------------------------------------------------------
// Rate limiter HTTP middleware
// ---------------------------------------------------------------------------

func TestRateLimiter_Handle_Disabled(t *testing.T) {
	t.Parallel()

	rl := newTestRateLimiter(t, 1, time.Minute)
	rl.config.Enabled = false
	rl.httpCore = &LimenHTTPCore{Responder: newTestResponder(t)}

	called := false
	handler := rl.handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.True(t, called)
}

func TestRateLimiter_Handle_AllowsWithinLimit(t *testing.T) {
	t.Parallel()

	rl := newTestRateLimiter(t, 5, time.Minute)
	rl.httpCore = &LimenHTTPCore{Responder: newTestResponder(t)}

	handler := rl.handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Handle_BlocksExceeded(t *testing.T) {
	t.Parallel()

	rl := newTestRateLimiter(t, 2, time.Minute)
	rl.httpCore = &LimenHTTPCore{Responder: newTestResponder(t)}

	handler := rl.handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for range 2 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimiter_Handle_DisabledRule(t *testing.T) {
	t.Parallel()

	rl := newTestRateLimiter(t, 1, time.Minute)
	rl.httpCore = &LimenHTTPCore{Responder: newTestResponder(t)}
	disabledRule := NewRateLimitRuleDisabledForPath("/health")
	disabledRule.pathRegex = regexp.MustCompile("^/health$")
	rl.rules = []*RateLimitRule{disabledRule}

	called := 0
	handler := rl.handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	}))

	for range 5 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
	assert.Equal(t, 5, called, "disabled rule should bypass rate limiting")
}
