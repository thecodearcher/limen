package admin

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func serveAs(t *testing.T, l *limen.Limen, user *limen.User, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	session := limen.SeedTestSession(t, l, user.ID, user.Email)
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(session.Cookie)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)
	return w
}

func TestGetUserHandler_PublicID(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPluginWithPublicIDs(t)
	admin := seedAdmin(t, l, plugin, "admin@test.com")
	target := seedAdmin(t, l, plugin, "target@test.com")
	publicID, ok := plugin.core.EncodePublicID(plugin.core.Schema.User, target)
	require.True(t, ok)

	w := serveAs(t, l, admin, http.MethodGet, "/auth/admin/users/"+publicID, "")

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		ID          string   `json:"id"`
		Roles       []string `json:"roles"`
		Permissions []string `json:"permissions"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, publicID, body.ID)
	assert.Equal(t, []string{"admin"}, body.Roles)
	assert.Contains(t, body.Permissions, "session:revoke")
}

func TestListRolesHandler(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name     string
		admin    bool
		wantCode int
	}{
		{name: "admin", admin: true, wantCode: http.StatusOK},
		{name: "member", admin: false, wantCode: http.StatusForbidden},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPlugin(t)
			user := limen.SeedTestUser(t, l, "user@test.com")
			if tt.admin {
				assignRole(t, plugin, user.ID, roleNameAdmin)
			}

			w := serveAs(t, l, user, http.MethodGet, "/auth/admin/roles", "")
			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestListUsersHandler_Filter(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	admin := seedAdmin(t, l, plugin, "admin@test.com")
	verified := limen.SeedTestUser(t, l, "verified@test.com")
	setUserColumns(t, plugin, verified.ID, map[limen.SchemaField]any{limen.UserSchemaEmailVerifiedAtField: time.Now()})

	w := serveAs(t, l, admin, http.MethodGet, "/auth/admin/users?filter[email_verified]=true", "")

	require.Equal(t, http.StatusOK, w.Code)
	var page struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	require.Len(t, page.Items, 1)
	assert.Equal(t, "verified@test.com", page.Items[0]["email"])
}

func TestListUsersHandler_UserShape(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name     string
		opt      ConfigOption
		wantKeys []string
	}{
		{name: "fields", opt: WithUserFields("id", "email"), wantKeys: []string{"id", "email", "roles", "permissions"}},
		{name: "serializer", opt: WithUserSerializer(func(user *limen.User) map[string]any {
			return map[string]any{"handle": user.Email}
		}), wantKeys: []string{"handle", "roles", "permissions"}},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPluginWithPublicIDs(t, tt.opt)
			admin := seedAdmin(t, l, plugin, "admin@test.com")

			w := serveAs(t, l, admin, http.MethodGet, "/auth/admin/users", "")

			require.Equal(t, http.StatusOK, w.Code)
			var page struct {
				Items []map[string]any `json:"items"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
			require.Len(t, page.Items, 1)
			assert.ElementsMatch(t, tt.wantKeys, slices.Collect(maps.Keys(page.Items[0])))
		})
	}
}

func TestSetRolesHandler_EmptyListClears(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPluginWithPublicIDs(t)
	admin := seedAdmin(t, l, plugin, "admin@test.com")
	target := seedAdmin(t, l, plugin, "target@test.com")
	publicID, ok := plugin.core.EncodePublicID(plugin.core.Schema.User, target)
	require.True(t, ok)

	w := serveAs(t, l, admin, http.MethodPut, "/auth/admin/users/"+publicID+"/roles", `{"roles": []}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Roles []string `json:"roles"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, []string{"user"}, body.Roles)
}

func TestListUsersHandler_Roles(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	admin := seedAdmin(t, l, plugin, "admin@test.com")
	limen.SeedTestUser(t, l, "member@test.com")

	w := serveAs(t, l, admin, http.MethodGet, "/auth/admin/users", "")

	require.Equal(t, http.StatusOK, w.Code)
	var page struct {
		Items []struct {
			Email string   `json:"email"`
			Roles []string `json:"roles"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	roles := make(map[string][]string, len(page.Items))
	for _, item := range page.Items {
		roles[item.Email] = item.Roles
	}
	assert.Equal(t, map[string][]string{"admin@test.com": {"admin"}, "member@test.com": {"user"}}, roles)
}

func TestBanUserHandler(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name     string
		opts     []ConfigOption
		wantCode int
	}{
		{name: "bans enabled", wantCode: http.StatusOK},
		{name: "bans disabled", opts: []ConfigOption{WithBans(false)}, wantCode: http.StatusNotFound},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPluginWithPublicIDs(t, tt.opts...)
			admin := seedAdmin(t, l, plugin, "admin@test.com")
			target := limen.SeedTestUser(t, l, "target@test.com")
			publicID, ok := plugin.core.EncodePublicID(plugin.core.Schema.User, target)
			require.True(t, ok)

			w := serveAs(t, l, admin, http.MethodPost, "/auth/admin/users/"+publicID+"/ban", `{"reason": "spam"}`)

			require.Equal(t, tt.wantCode, w.Code, w.Body.String())
			if tt.wantCode != http.StatusOK {
				return
			}
			var body struct {
				BannedAt  *time.Time `json:"banned_at"`
				BanReason string     `json:"ban_reason"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.NotNil(t, body.BannedAt)
			assert.Equal(t, "spam", body.BanReason)
		})
	}
}

func TestSetUserPasswordHandler_PasswordRequired(t *testing.T) {
	t.Parallel()

	plugin := New()
	l, _ := limen.NewTestLimenWithSchema(t, limen.NewDefaultSchemaConfig(limen.WithPublicIDs(testPublicIDConfig())), plugin, fakePasswordPlugin{})
	admin := seedAdmin(t, l, plugin, "admin@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	publicID, ok := plugin.core.EncodePublicID(plugin.core.Schema.User, target)
	require.True(t, ok)

	w := serveAs(t, l, admin, http.MethodPut, "/auth/admin/users/"+publicID+"/password", `{"revoke_sessions": true}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
}

func serveWithCookies(t *testing.T, l *limen.Limen, method, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)
	return w
}

func responseCookies(w *httptest.ResponseRecorder) map[string]*http.Cookie {
	cookies := make(map[string]*http.Cookie)
	for _, cookie := range w.Result().Cookies() {
		cookies[cookie.Name] = cookie
	}
	return cookies
}

func TestImpersonationHandlers(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPluginWithPublicIDs(t)
	admin := seedAdmin(t, l, plugin, "admin@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	publicID, ok := plugin.core.EncodePublicID(plugin.core.Schema.User, target)
	require.True(t, ok)
	adminCookie := limen.SeedTestSession(t, l, admin.ID, admin.Email).Cookie

	w := serveWithCookies(t, l, http.MethodPost, "/auth/admin/users/"+publicID+"/impersonate", adminCookie)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	cookies := responseCookies(w)
	impersonationCookie, stashCookie := cookies[adminCookie.Name], cookies[plugin.config.adminSessionCookieName]
	require.NotNil(t, impersonationCookie)
	require.NotNil(t, stashCookie)
	assert.NotEqual(t, adminCookie.Value, impersonationCookie.Value)

	w = serveWithCookies(t, l, http.MethodPost, "/auth/admin/impersonation/stop", impersonationCookie, stashCookie)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	cookies = responseCookies(w)
	require.NotNil(t, cookies[adminCookie.Name])
	assert.Equal(t, adminCookie.Value, cookies[adminCookie.Name].Value)
	require.NotNil(t, cookies[plugin.config.adminSessionCookieName])
	assert.Negative(t, cookies[plugin.config.adminSessionCookieName].MaxAge)
}
