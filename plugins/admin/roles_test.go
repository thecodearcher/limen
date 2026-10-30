package admin

import (
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func TestAdminPlugin_GetUserRoles(t *testing.T) {
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

			roles, err := plugin.GetUserRoles(t.Context(), user)
			require.NoError(t, err)
			assert.Equal(t, tt.want, roleNames(roles))
		})
	}
}

func TestAdminPlugin_GetUserRoles_AdminUserIDs(t *testing.T) {
	t.Parallel()

	_, plugin := newTestAdminPlugin(t, WithAdminUserIDs("root-admin"))
	err := plugin.core.DBAction.CreateUser(t.Context(), &limen.User{Email: "root@test.com"}, map[string]any{"id": "root-admin"})
	require.NoError(t, err)
	user, err := plugin.core.DBAction.FindUserByEmail(t.Context(), "root@test.com")
	require.NoError(t, err)

	roles, err := plugin.GetUserRoles(t.Context(), user)
	require.NoError(t, err)
	assert.Equal(t, []string{"user", "admin"}, roleNames(roles))
}

func TestAdminPlugin_GetUserRoles_AdminUserIDsPublicID(t *testing.T) {
	t.Parallel()

	_, plugin := newTestAdminPluginWithPublicIDs(t, WithAdminUserIDs("usr_root"))
	err := plugin.core.DBAction.CreateUser(t.Context(), &limen.User{Email: "root@test.com"}, map[string]any{"public_id": "usr_root"})
	require.NoError(t, err)
	user, err := plugin.core.DBAction.FindUserByEmail(t.Context(), "root@test.com")
	require.NoError(t, err)

	roles, err := plugin.GetUserRoles(t.Context(), user)
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

func newTestAdminPluginWithSupportRole(t *testing.T, opts ...ConfigOption) (*limen.Limen, *adminPlugin) {
	t.Helper()
	ac := access.New(DefaultStatements())
	roles, err := DefaultRoles(ac)
	require.NoError(t, err)
	support, err := ac.NewRole("support", access.P("user:set-role,list"))
	require.NoError(t, err)
	return newTestAdminPlugin(t, append([]ConfigOption{WithAccessControl(ac), WithRoles(append(roles, support)...)}, opts...)...)
}

func userRoleRows(t *testing.T, plugin *adminPlugin, userID any) map[string]any {
	t.Helper()
	models, err := plugin.core.FindMany(t.Context(), plugin.userRoleSchema, []limen.Where{
		limen.Eq(plugin.userRoleSchema.GetUserIDField(), userID),
	})
	require.NoError(t, err)
	rows := make(map[string]any, len(models))
	for _, userRole := range limen.MapToSliceOfType[*UserRole](models) {
		rows[userRole.Role] = userRole.ID
	}
	return rows
}

func TestAdminPlugin_SetRoles_AppliesDifference(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPluginWithSupportRole(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	assignRole(t, plugin, target.ID, roleNameUser)
	assignRole(t, plugin, target.ID, "support")
	supportRowID := userRoleRows(t, plugin, target.ID)["support"]

	_, err := plugin.SetRoles(t.Context(), actor, target.ID, []string{"support", "admin"})
	require.NoError(t, err)

	rows := userRoleRows(t, plugin, target.ID)
	assert.ElementsMatch(t, []string{"support", "admin"}, slices.Collect(maps.Keys(rows)))
	assert.Equal(t, supportRowID, rows["support"])
}

func TestAdminPlugin_SetRoles_UnknownRole(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")

	_, err := plugin.SetRoles(t.Context(), actor, target.ID, []string{"ghost"})
	var limenErr *limen.LimenError
	require.True(t, errors.As(err, &limenErr))
	assert.Equal(t, "unknown_role", limenErr.Code())
}

func TestAdminPlugin_SetRoles_LastAdmin(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	admin := seedAdmin(t, l, plugin, "admin@test.com")

	_, err := plugin.SetRoles(t.Context(), admin, admin.ID, []string{roleNameUser})
	assert.ErrorIs(t, err, ErrLastAdmin)
}

func TestAdminPlugin_SetRoles_Grantable(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name          string
		targetIsAdmin bool
		roles         []string
		wantErr       error
	}{
		{name: "add role", targetIsAdmin: false, roles: []string{roleNameAdmin}, wantErr: ErrRoleNotGrantable},
		{name: "remove role", targetIsAdmin: true, roles: []string{}, wantErr: ErrRoleNotGrantable},
		{name: "keep role", targetIsAdmin: true, roles: []string{roleNameAdmin, roleNameUser}, wantErr: nil},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPluginWithSupportRole(t)
			seedAdmin(t, l, plugin, "other-admin@test.com")
			actor := limen.SeedTestUser(t, l, "support@test.com")
			assignRole(t, plugin, actor.ID, "support")
			target := limen.SeedTestUser(t, l, "target@test.com")
			if tt.targetIsAdmin {
				assignRole(t, plugin, target.ID, roleNameAdmin)
			}

			_, err := plugin.SetRoles(t.Context(), actor, target.ID, tt.roles)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
