package admin

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func TestAdminPlugin_Initialize_InvalidRoles(t *testing.T) {
	t.Parallel()

	admin, err := DefaultAdminRole(access.New(DefaultStatements()))
	require.NoError(t, err)

	extendedAC := access.New(access.MergeStatements(DefaultStatements(), access.Statements{"project": {"read"}}))
	projectReader, err := extendedAC.NewRole("reader", access.P("project:read"))
	require.NoError(t, err)

	testData := []struct {
		name    string
		roles   []access.Role
		wantErr string
	}{
		{name: "duplicate role", roles: []access.Role{admin, admin}, wantErr: "defined multiple times"},
		{name: "unknown permission", roles: []access.Role{projectReader}, wantErr: "not valid against the configured access control"},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := New(WithRoles(tt.roles...)).Initialize(nil)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestAdminPlugin_Initialize_NoAccessControl(t *testing.T) {
	t.Parallel()

	err := New(WithAccessControl(nil)).Initialize(nil)
	assert.ErrorContains(t, err, "access control must be provided")
}

func TestAdminPlugin_Initialize_UnknownRoleReference(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name    string
		opt     ConfigOption
		wantErr string
	}{
		{name: "default role", opt: WithDefaultRoles("ghost"), wantErr: `default role "ghost"`},
		{name: "admin role", opt: WithAdminRoles("ghost"), wantErr: `admin role "ghost"`},
		{name: "secondary admin role", opt: WithAdminRoles(roleNameAdmin, "ghost"), wantErr: `admin role "ghost"`},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := New(tt.opt).Initialize(nil)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestAdminPlugin_BeforeSessionCreate(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name      string
		expiresAt time.Time
		wantErr   error
	}{
		{name: "permanent ban", wantErr: ErrUserBanned},
		{name: "active ban", expiresAt: time.Now().Add(time.Hour), wantErr: ErrUserBanned},
		{name: "expired ban", expiresAt: time.Now().Add(-time.Hour)},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPlugin(t)
			target := limen.SeedTestUser(t, l, "target@test.com")
			payload := map[limen.SchemaField]any{UserSchemaBannedAtField: time.Now()}
			if !tt.expiresAt.IsZero() {
				payload[UserSchemaBanExpiresAtField] = tt.expiresAt
			}
			require.NoError(t, plugin.core.Update(t.Context(), plugin.userSchema, payload, []limen.Where{
				limen.Eq(plugin.userSchema.GetIDField(), target.ID),
			}))
			before := banColumns(t, plugin, target.ID)

			err := createSession(t, plugin, target)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, before, banColumns(t, plugin, target.ID))
		})
	}
}

func TestAdminPlugin_BeforeSessionCreate_BannedMessage(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t, WithBannedMessage(func(user *limen.User) string {
		return user.Email + " is banned"
	}))
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	_, err := plugin.BanUser(t.Context(), actor, target.ID, &BanUserRequest{})
	require.NoError(t, err)

	err = createSession(t, plugin, target)

	var limenErr *limen.LimenError
	require.True(t, errors.As(err, &limenErr))
	assert.Equal(t, ErrUserBanned.Code(), limenErr.Code())
	assert.Equal(t, "target@test.com is banned", limenErr.Error())
}
