package admin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen/access"
)

func TestDefaultRoleMatrix(t *testing.T) {
	t.Parallel()

	roles, err := DefaultRoles(access.New(DefaultStatements()))
	require.NoError(t, err)
	admin, user := roles[0], roles[1]

	tests := []struct {
		role    access.Role
		name    string
		granted []string
		denied  []string
	}{
		{
			role:    admin,
			name:    "admin",
			granted: []string{"user:ban", "user:impersonate", "user:set-role", "session:revoke"},
			denied:  []string{"user:impersonate-admins", "user:set-password-admins", "user:set-email-admins"},
		},
		{
			role:   user,
			name:   "user",
			denied: []string{"user:read", "session:list"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.name, tt.role.Name())
			for _, perm := range tt.granted {
				resource, action, _ := strings.Cut(perm, ":")
				assert.Truef(t, tt.role.Can(resource, action), "%s should grant %s", tt.name, perm)
			}
			for _, perm := range tt.denied {
				resource, action, _ := strings.Cut(perm, ":")
				assert.Falsef(t, tt.role.Can(resource, action), "%s should not grant %s", tt.name, perm)
			}
		})
	}
}
