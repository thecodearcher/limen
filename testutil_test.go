package limen

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestLimen(t *testing.T, plugins ...Plugin) *Limen {
	t.Helper()
	l, _ := NewTestLimen(t, plugins...)
	return l
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
		Secret:   TestSecret,
		Session:  NewDefaultSessionConfig(opts...),
	})
	require.NoError(t, err)
	return l
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

func (p *testPlugin) GetSchemas(schema *SchemaConfig) []SchemaIntrospector {
	return []SchemaIntrospector{}
}

func (p *testPlugin) RegisterRoutes(httpCore *LimenHTTPCore, routeBuilder *RouteBuilder) {
	routeBuilder.AddRoute(MethodGET, "/test", RouteID("test"), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, nil, nil)
}

func (p *testPlugin) TestMethodOnPlugin() string {
	return "test-method-on-plugin"
}
