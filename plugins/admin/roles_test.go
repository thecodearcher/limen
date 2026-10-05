package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func TestAdminPlugin_GetRoles(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name     string
		assigned []string
		want     []string
	}{
		{name: "no assigned roles uses defaults", assigned: nil, want: []string{"user"}},
		{name: "assigned roles replace defaults", assigned: []string{"admin"}, want: []string{"admin"}},
		{name: "unconfigured assigned role is ignored", assigned: []string{"ghost"}, want: []string{}},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPlugin(t)
			user := limen.SeedTestUser(t, l, "roles@test.com")
			for _, role := range tt.assigned {
				assignRole(t, plugin, user.ID, role)
			}

			roles, err := plugin.GetRoles(t.Context(), user)
			require.NoError(t, err)
			assert.Equal(t, tt.want, roleNames(roles))
		})
	}
}

func TestAdminPlugin_GetRoles_AdminUserIDs(t *testing.T) {
	t.Parallel()

	_, plugin := newTestAdminPlugin(t, WithAdminUserIDs("root-admin"))
	err := plugin.core.DBAction.CreateUser(t.Context(), &limen.User{Email: "root@test.com"}, map[string]any{"id": "root-admin"})
	require.NoError(t, err)
	user, err := plugin.core.DBAction.FindUserByEmail(t.Context(), "root@test.com")
	require.NoError(t, err)

	roles, err := plugin.GetRoles(t.Context(), user)
	require.NoError(t, err)
	assert.Equal(t, []string{"user", "admin"}, roleNames(roles))
}

func TestAdminPlugin_GetRoles_AdminUserIDsPublicID(t *testing.T) {
	t.Parallel()

	_, plugin := newTestAdminPluginWithPublicIDs(t, WithAdminUserIDs("usr_root"))
	err := plugin.core.DBAction.CreateUser(t.Context(), &limen.User{Email: "root@test.com"}, map[string]any{"public_id": "usr_root"})
	require.NoError(t, err)
	user, err := plugin.core.DBAction.FindUserByEmail(t.Context(), "root@test.com")
	require.NoError(t, err)

	roles, err := plugin.GetRoles(t.Context(), user)
	require.NoError(t, err)
	assert.Equal(t, []string{"user", "admin"}, roleNames(roles))
}

func TestAdminPlugin_HasPermission(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	member := limen.SeedTestUser(t, l, "member@test.com")
	admin := limen.SeedTestUser(t, l, "admin@test.com")
	assignRole(t, plugin, admin.ID, "admin")

	assert.ErrorIs(t, plugin.HasPermission(t.Context(), member, access.P("user:ban")), ErrInsufficientPermission)
	assert.NoError(t, plugin.HasPermission(t.Context(), admin, access.P("user:ban")))
}
