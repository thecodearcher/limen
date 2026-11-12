package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAC(t *testing.T) *AccessControl {
	t.Helper()
	return New(Statements{
		"organization": {"create", "read", "update", "delete"},
		"project":      {"create", "read"},
	})
}

func newTestRole(t *testing.T, ac *AccessControl, name string, perms Permissions) Role {
	t.Helper()
	role, err := ac.NewRole(name, perms)
	require.NoError(t, err)
	return role
}

func TestNewRole(t *testing.T) {
	t.Parallel()

	ac := newTestAC(t)

	tests := []struct {
		name     string
		roleName string
		perms    Permissions
		wantErr  string
	}{
		{
			name:     "valid grants",
			roleName: "editor",
			perms:    Permissions{"organization": {"read"}, "project": {"create", "read"}},
		},
		{
			name:     "wildcard is always valid",
			roleName: "owner",
			perms:    Permissions{"organization": {"*"}},
		},
		{
			name:     "empty grants",
			roleName: "bystander",
			perms:    Permissions{},
		},
		{
			name:     "empty role name",
			roleName: "",
			perms:    Permissions{},
			wantErr:  "role name must not be empty",
		},
		{
			name:     "unknown resource",
			roleName: "editor",
			perms:    Permissions{"porject": {"read"}},
			wantErr:  `role "editor": unknown resource "porject"`,
		},
		{
			name:     "unknown action",
			roleName: "editor",
			perms:    Permissions{"project": {"delete"}},
			wantErr:  `role "editor": unknown action "delete" for resource "project"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			role, err := ac.NewRole(tt.roleName, tt.perms)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.roleName, role.Name())
		})
	}
}

func TestNewRoleCopiesInput(t *testing.T) {
	t.Parallel()

	ac := newTestAC(t)
	input := Permissions{"project": {"read"}}
	role := newTestRole(t, ac, "viewer", input)

	input["project"][0] = "create"

	assert.True(t, role.Can("project", "read"))
	assert.False(t, role.Can("project", "create"))
}

func TestRoleCan(t *testing.T) {
	t.Parallel()

	ac := newTestAC(t)
	editor := newTestRole(t, ac, "editor", Permissions{
		"organization": {"read", "update"},
		"project":      {"*"},
	})

	tests := []struct {
		name             string
		resource, action string
		want             bool
	}{
		{"granted action", "organization", "read", true},
		{"second granted action", "organization", "update", true},
		{"ungranted action", "organization", "delete", false},
		{"wildcard grant", "project", "create", true},
		// Wildcard matches lazily: actions never declared still match, so
		// default roles built against a plugin's own statements keep working
		// when the app extends the resource with new actions.
		{"wildcard matches undeclared action", "project", "archive", true},
		{"unknown resource", "report", "read", false},
		{"empty resource", "", "read", false},
		{"empty action", "organization", "", false},
		{"empty action with wildcard grant", "project", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, editor.Can(tt.resource, tt.action))
		})
	}
}

func TestRoleCanAll(t *testing.T) {
	t.Parallel()

	ac := newTestAC(t)
	editor := newTestRole(t, ac, "editor", Permissions{
		"organization": {"read"},
		"project":      {"*"},
	})

	assert.True(t, editor.CanAll(Permissions{"organization": {"read"}, "project": {"create", "read"}}))
	assert.False(t, editor.CanAll(Permissions{"organization": {"read", "update"}}))
	assert.False(t, editor.CanAll(Permissions{}))
}

func TestRolePermissionsReturnsCopy(t *testing.T) {
	t.Parallel()

	ac := newTestAC(t)
	editor := newTestRole(t, ac, "editor", Permissions{"organization": {"read"}})

	perms := editor.Permissions()
	perms["organization"][0] = "delete"
	perms["injected"] = []string{"boom"}

	assert.Equal(t, Permissions{"organization": {"read"}}, editor.Permissions())
	assert.False(t, editor.Can("organization", "delete"))
}

func TestMergeRolePermissions(t *testing.T) {
	t.Parallel()

	ac := New(Statements{"project": {"read", "create"}, "member": {"read"}})
	reader, err := ac.NewRole("reader", Permissions{"project": {"read"}})
	require.NoError(t, err)
	editor, err := ac.NewRole("editor", Permissions{"project": {"create"}, "member": {"read"}})
	require.NoError(t, err)

	assert.Equal(t, Permissions{"project": {"read", "create"}, "member": {"read"}}, MergeRolePermissions(reader, editor))
}
