package admin

import "github.com/thecodearcher/limen/access"

type config struct {
	accessControl        *access.AccessControl
	roles                []access.Role
	defaultRoles         []string
	adminRole            string
	adminUserIDs         []any
	bansEnabled          bool
	impersonationEnabled bool
}

type ConfigOption func(*config)

// WithAccessControl sets the vocabulary roles are validated against.
func WithAccessControl(accessControl *access.AccessControl) ConfigOption {
	return func(c *config) {
		c.accessControl = accessControl
	}
}

// WithRoles sets the roles that can be assigned to users. Defaults to DefaultRoles.
func WithRoles(roles ...access.Role) ConfigOption {
	return func(c *config) {
		c.roles = roles
	}
}

// WithDefaultRoles sets the roles a user has when no roles are assigned. Defaults to "user".
func WithDefaultRoles(names ...string) ConfigOption {
	return func(c *config) {
		c.defaultRoles = names
	}
}

// WithAdminRole sets the role that counts as admin. Defaults to "admin".
func WithAdminRole(name string) ConfigOption {
	return func(c *config) {
		c.adminRole = name
	}
}

// WithAdminUserIDs grants the admin role to these users on top of their assigned roles.
func WithAdminUserIDs(ids ...any) ConfigOption {
	return func(c *config) {
		c.adminUserIDs = ids
	}
}

// WithBans toggles user bans and their columns on the users table.
func WithBans(enabled bool) ConfigOption {
	return func(c *config) {
		c.bansEnabled = enabled
	}
}

// WithImpersonation toggles impersonation and its column on the sessions table.
func WithImpersonation(enabled bool) ConfigOption {
	return func(c *config) {
		c.impersonationEnabled = enabled
	}
}
