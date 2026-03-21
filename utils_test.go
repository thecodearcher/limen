package limen

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString_Length(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		length int
	}{
		{"zero", 0},
		{"one", 1},
		{"short", 8},
		{"long", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateRandomString(tt.length)
			assert.Len(t, result, tt.length)
		})
	}
}

func TestGenerateRandomString_Alphanumeric(t *testing.T) {
	t.Parallel()

	result := GenerateRandomString(1000, CharSetAlphanumeric)
	for _, c := range result {
		assert.True(t,
			(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'),
			"unexpected character: %c", c)
	}
}

func TestGenerateRandomString_Numeric(t *testing.T) {
	t.Parallel()

	result := GenerateRandomString(1000, CharSetNumeric)
	for _, c := range result {
		assert.True(t, c >= '0' && c <= '9', "expected digit, got: %c", c)
	}
}

func TestGetFromMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected string
	}{
		{"found", map[string]any{"name": "Alice"}, "name", "Alice"},
		{"missing key", map[string]any{"name": "Alice"}, "missing", ""},
		{"type mismatch", map[string]any{"age": 42}, "age", ""},
		{"nil map", nil, "key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetFromMap[string](tt.m, tt.key))
		})
	}
}

func TestSortRulesBySpecificity(t *testing.T) {
	t.Parallel()

	rules := []*RateLimitRule{
		NewRateLimitRule("/api/*", 10, time.Minute),
		NewRateLimitRule("/api/users", 20, time.Minute),
		NewRateLimitRule("/api/users/:id", 30, time.Minute),
		NewRateLimitRule("**", 5, time.Minute),
	}

	sortRulesBySpecificity(rules)

	assert.Equal(t, "/api/users", rules[0].path, "exact match should be first")
	assert.Equal(t, "/api/users/:id", rules[1].path, "parameterized path should be next")
	assert.Equal(t, "/api/*", rules[2].path, "single wildcard path")
	assert.Equal(t, "**", rules[3].path, "double wildcard should be last")
}

func TestNormalizePluginPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		basePath   string
		pluginPath string
		override   *PluginHTTPOverride
		expected   string
	}{
		{
			name:       "basic join",
			basePath:   "/api/auth",
			pluginPath: "/credential-password",
			expected:   "/api/auth/credential-password",
		},
		{
			name:       "override replaces plugin path",
			basePath:   "/api/auth",
			pluginPath: "/credential-password",
			override:   &PluginHTTPOverride{BasePath: "/custom"},
			expected:   "/api/auth/custom",
		},
		{
			name:       "empty override base path keeps plugin path",
			basePath:   "/api/auth",
			pluginPath: "/oauth",
			override:   &PluginHTTPOverride{BasePath: ""},
			expected:   "/api/auth/oauth",
		},
		{
			name:       "nil override uses plugin path",
			basePath:   "/api",
			pluginPath: "/session",
			override:   nil,
			expected:   "/api/session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePluginPath(tt.basePath, tt.pluginPath, tt.override)
			assert.Equal(t, tt.expected, result)
		})
	}
}
