package admin

import (
	"github.com/thecodearcher/limen/access"
)

const (
	roleNameAdmin = "admin"
	roleNameUser  = "user"
)

var defaultStatements = access.Statements{
	"user":    {"create", "list", "read", "update", "delete", "ban", "impersonate", "impersonate-admins", "set-password", "set-password-admins", "set-role", "set-email", "set-email-admins"},
	"session": {"list", "revoke"},
}

// DefaultStatements returns a copy of the plugin's built-in vocabulary.
func DefaultStatements() access.Statements {
	return access.MergeStatements(defaultStatements)
}

// DefaultAdminRole builds the built-in admin role against ac. It can manage every
// user.
func DefaultAdminRole(ac *access.AccessControl, extra ...access.Permissions) (access.Role, error) {
	return ac.NewRole(roleNameAdmin, access.MergePermissions(append([]access.Permissions{{
		"user":    {"create", "list", "read", "update", "delete", "ban", "impersonate", "set-password", "set-role", "set-email"},
		"session": {"*"},
	}}, extra...)...))
}

// DefaultUserRole builds the built-in user role against ac.
func DefaultUserRole(ac *access.AccessControl, extra ...access.Permissions) (access.Role, error) {
	return ac.NewRole(roleNameUser, access.MergePermissions(extra...))
}

// DefaultRoles builds the built-in roles against ac.
func DefaultRoles(ac *access.AccessControl) ([]access.Role, error) {
	builders := []func(*access.AccessControl, ...access.Permissions) (access.Role, error){
		DefaultAdminRole, DefaultUserRole,
	}

	roles := make([]access.Role, 0, len(builders))

	for _, builder := range builders {
		role, err := builder(ac)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}
