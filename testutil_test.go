package limen

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestLimen(t *testing.T, plugins ...Plugin) *Limen {
	t.Helper()
	l, _ := NewTestLimen(t, plugins...)
	return l
}

func ptr[T any](value T) *T {
	return &value
}

func seedUser(t *testing.T, l *Limen, email string) any {
	t.Helper()
	return SeedTestUser(t, l, email).ID
}

func seedSession(t *testing.T, l *Limen, userID any, email string) *SessionResult {
	t.Helper()
	return SeedTestSession(t, l, userID, email)
}

// ---------------------------------------------------------------------------
// Core-specific helpers (not shared with plugins)
// ---------------------------------------------------------------------------

func newTestLimenWithSessionConfig(t *testing.T, opts ...SessionConfigOption) *Limen {
	t.Helper()

	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: newTestMemoryAdapter(t),
		Secret:   testSecret,
		Session:  NewDefaultSessionConfig(opts...),
	})
	require.NoError(t, err)
	return l
}

func jsonRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func copyResponseCookies(t *testing.T, rec *httptest.ResponseRecorder, req *http.Request) {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		req.AddCookie(cookie)
	}
}

func findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	var found *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			found = cookie
		}
	}
	return found
}

func newTestHTTPCore(t *testing.T, l *Limen) *LimenHTTPCore {
	t.Helper()
	return &LimenHTTPCore{
		Responder:              newResponder(l.config.HTTP, l.core.cookies, l.config.Session.BearerEnabled),
		authInstance:           l,
		core:                   l.core,
		config:                 l.config.HTTP,
		trustedOriginsPatterns: []*regexp.Regexp{},
	}
}

// ---------------------------------------------------------------------------
// Test plugin
// ---------------------------------------------------------------------------

func newTestPlugin(t *testing.T) Plugin {
	t.Helper()
	return &testPlugin{}
}

type testPlugin struct{}

func (p *testPlugin) Name() PluginName {
	return "test"
}

func (p *testPlugin) Initialize(core *LimenCore) error {
	return nil
}

func (p *testPlugin) PluginHTTPConfig() PluginHTTPConfig {
	return PluginHTTPConfig{
		BasePath: "/test",
	}
}

func (p *testPlugin) RegisterRoutes(httpCore *LimenHTTPCore, routeBuilder *RouteBuilder) {
	routeBuilder.AddRoute(MethodGET, "/test", RouteID("test"), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, nil, nil)
}

func (p *testPlugin) TestMethodOnPlugin() string {
	return "test-method-on-plugin"
}
