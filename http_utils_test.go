package limen

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "already normalized", input: "/auth", want: "/auth"},
		{name: "missing leading slash", input: "auth", want: "/auth"},
		{name: "trailing slash", input: "/auth/", want: "/auth"},
		{name: "both issues", input: "auth/", want: "/auth"},
		{name: "root", input: "/", want: ""},
		{name: "nested path", input: "/api/v1/auth/", want: "/api/v1/auth"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizePath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOriginMatcher(t *testing.T) {
	t.Parallel()

	patterns := compileTrustedOrigins("http://localhost:3000", "https://*.example.com")

	tests := []struct {
		name    string
		origin  string
		want    bool
		referer string
	}{
		{name: "exact match", origin: "http://localhost:3000", want: true},
		{name: "wildcard subdomain", origin: "https://app.example.com", want: true},
		{name: "no match", origin: "http://evil.com", want: false},
		{name: "empty origin", origin: "", want: false},
		{name: "referer match", origin: "", referer: "http://localhost:3000", want: true},
		{name: "referer no match", origin: "", referer: "http://evil.com", want: false},
		{name: "referer empty", origin: "", referer: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", http.NoBody)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			got := originMatcher(req, patterns)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJoinURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		base  string
		parts []string
		want  string
	}{
		{name: "simple join", base: "http://localhost:8080", parts: []string{"/auth"}, want: "http://localhost:8080/auth"},
		{name: "multiple parts", base: "http://localhost:8080", parts: []string{"/auth", "/signin"}, want: "http://localhost:8080/auth/signin"},
		{name: "trailing slashes normalized", base: "http://localhost:8080/", parts: []string{"/auth/"}, want: "http://localhost:8080/auth/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := joinURL(tt.base, tt.parts...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractCookieValue(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
	http.SetCookie(w, &http.Cookie{Name: "other", Value: "xyz"})

	val := ExtractCookieValue(w.Header(), "session")
	assert.Equal(t, "abc123", val)

	val = ExtractCookieValue(w.Header(), "nonexistent")
	assert.Equal(t, "", val)
}

func TestGlobToRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		pattern               string
		matches               []string
		noMatches             []string
		supportPathParameters bool
	}{
		// exact literal
		{
			name:      "exact",
			pattern:   "/auth/signin",
			matches:   []string{"/auth/signin"},
			noMatches: []string{"/auth/signup", "/auth", "/auth/signin/extra"},
		},
		{
			name:      "port",
			pattern:   "test:8080",
			matches:   []string{"test:8080"},
			noMatches: []string{"test:1080", "test"},
		},
		// * matches one segment (no slashes)
		{
			name:      "single star",
			pattern:   "/auth/*",
			matches:   []string{"/auth/signin", "/auth/signup", "/auth/me", "/auth/"},
			noMatches: []string{"/auth/oauth/google"},
		},
		// ** matches any depth including slashes
		{
			name:    "double star",
			pattern: "/auth/**",
			matches: []string{"/auth/oauth/google/callback", "/auth/a/b/c", "/auth/signin", "/auth/"},
		},
		// ? matches single character (not slash)
		{
			name:      "question mark",
			pattern:   "/auth/m?",
			matches:   []string{"/auth/me"},
			noMatches: []string{"/auth/m", "/auth/m/", "/auth/mee"},
		},
		{
			name:    "multiple question marks",
			pattern: "/api/v?/auth",
			matches: []string{"/api/v1/auth", "/api/v2/auth"},
		},
		// :param matches one path segment
		{
			name:                  "route param mid-path",
			pattern:               "/oauth/:provider/callback",
			supportPathParameters: true,
			matches:               []string{"/oauth/google/callback", "/oauth/github/callback"},
			noMatches:             []string{"/oauth/google/bad/callback", "/oauth//callback"},
		},
		{
			name:                  "route param at end",
			pattern:               "/users/:id",
			supportPathParameters: true,
			matches:               []string{"/users/42", "/users/abc"},
			noMatches:             []string{"/users/", "/users/42/edit"},
		},
		{
			name:                  "multiple route params",
			pattern:               "/api/:version/:resource",
			supportPathParameters: true,
			matches:               []string{"/api/v1/users", "/api/v2/posts"},
		},
		// [...] character class
		{
			name:      "character class",
			pattern:   "/api/v[12]/auth",
			matches:   []string{"/api/v1/auth", "/api/v2/auth"},
			noMatches: []string{"/api/v3/auth", "/api/v9/auth"},
		},
		// backslash escape
		{
			name:      "escaped star is literal",
			pattern:   `/auth/\*`,
			matches:   []string{"/auth/*"},
			noMatches: []string{"/auth/signin"},
		},
		// regex special characters are escaped
		{
			name:      "dot is literal",
			pattern:   "/auth/file.txt",
			matches:   []string{"/auth/file.txt"},
			noMatches: []string{"/auth/fileXtxt"},
		},
		{
			name:    "plus is literal",
			pattern: "/auth/c++",
			matches: []string{"/auth/c++"},
		},
		{
			name:    "parens are literal",
			pattern: "/auth/(group)",
			matches: []string{"/auth/(group)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			regex := globToRegex(tt.pattern, tt.supportPathParameters)
			re := regexp.MustCompile(regex)

			for _, input := range tt.matches {
				assert.True(t, re.MatchString(input), "expected %q to match %q (regex: %s)", tt.pattern, input, regex)
			}
			for _, input := range tt.noMatches {
				assert.False(t, re.MatchString(input), "expected %q NOT to match %q (regex: %s)", tt.pattern, input, regex)
			}
		})
	}
}

func TestIsValidCoreSchema(t *testing.T) {
	t.Parallel()

	assert.True(t, IsValidCoreSchema("users"))
	assert.True(t, IsValidCoreSchema("sessions"))
	assert.True(t, IsValidCoreSchema("verifications"))
	assert.True(t, IsValidCoreSchema("rate_limits"))
	assert.True(t, IsValidCoreSchema("accounts"))
	assert.False(t, IsValidCoreSchema("unknown"))
	assert.False(t, IsValidCoreSchema(""))
}
