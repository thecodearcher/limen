package admin

import (
	"context"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type API interface {
	// ListRoles returns each configured role's permissions with wildcards expanded to the declared actions. Requires user:set-role.
	ListRoles(ctx context.Context, actor *limen.User) (map[string]access.Permissions, error)
	// GetUserRoles returns the user's assigned roles, or the default roles when none are assigned.
	GetUserRoles(ctx context.Context, user *limen.User) ([]access.Role, error)
	// GetPermissions returns the union of the user's role permissions.
	GetPermissions(ctx context.Context, user *limen.User) (access.Permissions, error)
	// HasPermission returns ErrInsufficientPermission unless the user holds every permission.
	HasPermission(ctx context.Context, user *limen.User, permissions access.Permissions) error
	// SetRoles replaces the user's assigned roles and returns the user with their resolved roles.
	// An empty list removes them all, so the user falls back to the default roles. Requires
	// user:set-role, and the actor must hold every permission of each role added or removed.
	SetRoles(ctx context.Context, actor *limen.User, userID any, roles []string) (*UserWithRoles, error)

	// ListUsers returns a page of users matching query. Requires user:list.
	ListUsers(ctx context.Context, actor *limen.User, query *ListUsersQuery, opts *limen.QueryOptions) (*limen.Page[*limen.User], error)
	// ListUsersWithRoles returns a page of users matching query with their resolved roles. Requires user:list.
	ListUsersWithRoles(ctx context.Context, actor *limen.User, query *ListUsersQuery, opts *limen.QueryOptions) (*limen.Page[*UserWithRoles], error)
	// GetUser returns a user by ID or public ID. Requires user:read.
	GetUser(ctx context.Context, actor *limen.User, userID any) (*limen.User, error)
	// GetUserWithRoles returns a user with their resolved roles. Requires user:read.
	GetUserWithRoles(ctx context.Context, actor *limen.User, userID any) (*UserWithRoles, error)
	// CreateUser creates a user without a password. Requires user:create.
	CreateUser(ctx context.Context, actor *limen.User, req *CreateUserRequest) (*limen.User, error)
	// UpdateUser updates a user's profile fields. Requires user:update.
	UpdateUser(ctx context.Context, actor *limen.User, userID any, req *UpdateUserRequest) (*limen.User, error)
	// DeleteUser deletes a user and their sessions and accounts. Requires user:delete.
	DeleteUser(ctx context.Context, actor *limen.User, userID any) error
}

// Use returns the admin plugin API. Panics if the plugin is not registered.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginAdmin)
}

var _ API = (*adminPlugin)(nil)
