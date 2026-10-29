package admin

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func serveAs(t *testing.T, l *limen.Limen, user *limen.User, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	session := limen.SeedTestSession(t, l, user.ID, user.Email)
	req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
	req.AddCookie(session.Cookie)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)
	return w
}

func TestGetUserHandler_PublicID(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPluginWithPublicIDs(t)
	admin := seedAdmin(t, l, plugin, "admin@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	publicID, ok := plugin.core.EncodePublicID(plugin.core.Schema.User, target)
	require.True(t, ok)

	w := serveAs(t, l, admin, http.MethodGet, "/auth/admin/users/"+publicID)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, publicID, body["id"])
	assert.Equal(t, "target@test.com", body["email"])
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

			w := serveAs(t, l, user, http.MethodGet, "/auth/admin/roles")
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

	w := serveAs(t, l, admin, http.MethodGet, "/auth/admin/users?filter[email_verified]=true")

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
		{name: "fields", opt: WithUserFields("id", "email"), wantKeys: []string{"id", "email"}},
		{name: "serializer", opt: WithUserSerializer(func(user *limen.User) map[string]any {
			return map[string]any{"handle": user.Email}
		}), wantKeys: []string{"handle"}},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPluginWithPublicIDs(t, tt.opt)
			admin := seedAdmin(t, l, plugin, "admin@test.com")

			w := serveAs(t, l, admin, http.MethodGet, "/auth/admin/users")

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
