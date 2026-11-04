package sessionjwt

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

const testSessionTagField limen.SchemaField = "tag"

func newTestJWTLimen(t *testing.T, plugins ...limen.Plugin) (*limen.Limen, *limen.LimenCore, *sessionJWTPlugin) {
	t.Helper()
	plugin := New()
	l, core := limen.NewTestLimen(t, append([]limen.Plugin{plugin}, plugins...)...)
	return l, core, plugin
}

func newSessionTagPlugin(t *testing.T) limen.Plugin {
	t.Helper()
	return &sessionTagPlugin{}
}

type sessionTagPlugin struct{}

func (p *sessionTagPlugin) Name() limen.PluginName {
	return "session-tag-test"
}

func (p *sessionTagPlugin) Initialize(*limen.LimenCore) error {
	return nil
}

func (p *sessionTagPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{}
}

func (p *sessionTagPlugin) RegisterRoutes(*limen.LimenHTTPCore, *limen.RouteBuilder) {}

func (p *sessionTagPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	return []limen.SchemaIntrospector{
		limen.NewSchemaDefinitionForExtension(limen.CoreSchemaSessions, schema.Session,
			limen.WithSchemaField(testSessionTagField, limen.ColumnTypeString, limen.WithNullable(true)),
		),
	}
}

func TestJWTSessionManager_CreateSession_ExpiresIn(t *testing.T) {
	t.Parallel()

	l, core, plugin := newTestJWTLimen(t)
	user := limen.SeedTestUser(t, l, "jwt-expires@test.com")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", http.NoBody)

	result, err := core.SessionManager.CreateSession(t.Context(), req, &limen.AuthenticationResult{User: user},
		limen.WithSessionExpiresIn(5*time.Minute),
	)
	require.NoError(t, err)

	claims, err := plugin.VerifyAccessToken(result.Token)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), claims.ExpiresAt.Time, 2*time.Second)

	rawRefresh, _ := parseRefreshTokenValue(result.RefreshToken)
	rt, err := plugin.FindRefreshTokenByToken(t.Context(), rawRefresh)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), rt.ExpiresAt, 2*time.Second)
}

func TestJWTSessionManager_CreateSession_Data(t *testing.T) {
	t.Parallel()

	l, core, plugin := newTestJWTLimen(t, newSessionTagPlugin(t))
	user := limen.SeedTestUser(t, l, "jwt-data@test.com")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", http.NoBody)

	result, err := core.SessionManager.CreateSession(t.Context(), req, &limen.AuthenticationResult{User: user},
		limen.WithSessionData(map[limen.SchemaField]any{
			testSessionTagField: limen.SessionValue{Internal: "internal", Client: "client"},
		}),
	)
	require.NoError(t, err)

	claims, err := plugin.VerifyAccessToken(result.Token)
	require.NoError(t, err)
	assert.Equal(t, "client", claims.Session[string(testSessionTagField)])

	rawRefresh, _ := parseRefreshTokenValue(result.RefreshToken)
	rt, err := plugin.FindRefreshTokenByToken(t.Context(), rawRefresh)
	require.NoError(t, err)
	assert.Equal(t, "client", rt.SessionData[string(testSessionTagField)])
}

func TestJWTSessionManager_RevokeSession_ListedToken(t *testing.T) {
	t.Parallel()

	l, core, plugin := newTestJWTLimen(t)
	user := limen.SeedTestUser(t, l, "jwt-revoke@test.com")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", http.NoBody)
	_, err := core.SessionManager.CreateSession(t.Context(), req, &limen.AuthenticationResult{User: user})
	require.NoError(t, err)

	sessions, err := core.SessionManager.ListSessions(t.Context(), user.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	require.NoError(t, core.SessionManager.RevokeSession(t.Context(), sessions[0].Token))

	_, err = plugin.FindRefreshTokenByToken(t.Context(), sessions[0].Token)
	assert.Error(t, err)
}
