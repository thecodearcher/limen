package access

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Role is a named, immutable set of granted permissions.
type Role struct {
	id          any
	name        string
	permissions Permissions
}

// NewRole builds a role with the grants validated against the AccessControl's
// statements. A "*" action grants every action on its resource, including
// actions declared later.
func (ac *AccessControl) NewRole(name string, perms Permissions) (Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Role{}, errors.New("access: role name must not be empty")
	}

	for resource, actions := range perms {
		declared, ok := ac.statements[resource]
		if !ok {
			return Role{}, fmt.Errorf("access: role %q: unknown resource %q", name, resource)
		}
		for _, action := range actions {
			if action == Wildcard {
				continue
			}
			if !slices.Contains(declared, action) {
				return Role{}, fmt.Errorf("access: role %q: unknown action %q for resource %q", name, action, resource)
			}
		}
	}

	return Role{name: name, permissions: MergePermissions(perms)}, nil
}

func (ac *AccessControl) NewRoleWithID(id any, name string, perms Permissions) (Role, error) {
	r, err := ac.NewRole(name, perms)
	if err != nil {
		return Role{}, err
	}
	r.id = id
	return r, nil
}

func (r Role) ID() any {
	return r.id
}

// Name returns the role's name.
func (r Role) Name() string {
	return strings.ToLower(r.name)
}

// Can reports whether the role grants action on resource.
func (r Role) Can(resource, action string) bool {
	if resource == "" || action == "" {
		return false
	}
	return permits(r.permissions, resource, action)
}

// CanAll reports whether the role grants every listed permission.
func (r Role) CanAll(permissions Permissions) bool {
	if len(permissions) == 0 {
		return false
	}
	return HasRequiredPermissions(r.permissions, permissions)
}

// Permissions returns a copy of the role's grants.
func (r Role) Permissions() Permissions {
	return MergePermissions(r.permissions)
}

// MergeRolePermissions returns the union of the roles' permissions.
func MergeRolePermissions(roles ...Role) Permissions {
	perms := make([]Permissions, 0, len(roles))
	for _, role := range roles {
		perms = append(perms, role.permissions)
	}
	return MergePermissions(perms...)
}

func permits(perms Permissions, resource, action string) bool {
	actions, ok := perms[resource]
	if !ok {
		return false
	}
	if slices.Contains(actions, Wildcard) {
		return true
	}
	return slices.Contains(actions, action)
}
