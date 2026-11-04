package admin

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func newTestAdminPluginWithPasswords(t *testing.T) (*limen.Limen, *adminPlugin) {
	t.Helper()
	plugin := New()
	l, _ := limen.NewTestLimen(t, plugin, fakePasswordPlugin{})
	return l, plugin
}

func TestAdminPlugin_SetUserPassword(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name           string
		revokeSessions bool
		wantSessions   int
	}{
		{name: "keeps sessions", revokeSessions: false, wantSessions: 1},
		{name: "revokes sessions", revokeSessions: true, wantSessions: 0},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPluginWithPasswords(t)
			actor := seedAdmin(t, l, plugin, "actor@test.com")
			target := limen.SeedTestUser(t, l, "target@test.com")
			limen.SeedTestSession(t, l, target.ID, target.Email)

			err := plugin.SetUserPassword(t.Context(), actor, target.ID, &SetUserPasswordRequest{
				Password:       "new-password",
				RevokeSessions: tt.revokeSessions,
			})
			require.NoError(t, err)

			stored, err := plugin.core.DBAction.FindUserByID(t.Context(), target.ID)
			require.NoError(t, err)
			require.NotNil(t, stored.Password)
			assert.Equal(t, "hashed:new-password", *stored.Password)

			sessions, err := plugin.core.SessionManager.ListSessions(t.Context(), target.ID)
			require.NoError(t, err)
			assert.Len(t, sessions, tt.wantSessions)
		})
	}
}

func TestAdminPlugin_SetUserPassword_PolicyRejected(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPluginWithPasswords(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	before, err := plugin.core.DBAction.FindUserByID(t.Context(), target.ID)
	require.NoError(t, err)

	err = plugin.SetUserPassword(t.Context(), actor, target.ID, &SetUserPasswordRequest{Password: "short"})
	assert.ErrorIs(t, err, errFakeWeakPassword)

	after, err := plugin.core.DBAction.FindUserByID(t.Context(), target.ID)
	require.NoError(t, err)
	assert.Equal(t, before.Password, after.Password)
}

func TestAdminPlugin_SetUserPassword_PasswordsNotEnabled(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")

	err := plugin.SetUserPassword(t.Context(), actor, target.ID, &SetUserPasswordRequest{Password: "new-password"})
	assert.ErrorIs(t, err, ErrPasswordsNotEnabled)
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
