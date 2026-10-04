package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		{name: "admin role", opt: WithAdminRole("ghost"), wantErr: `admin role "ghost"`},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := New(tt.opt).Initialize(nil)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
