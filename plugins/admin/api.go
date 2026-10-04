package admin

import (
	"context"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type API interface {
	// GetRoles returns the user's assigned roles, or the default roles when none are assigned.
	GetRoles(ctx context.Context, user *limen.User) ([]access.Role, error)
	// GetPermissions returns the union of the user's role permissions.
	GetPermissions(ctx context.Context, user *limen.User) (access.Permissions, error)
	// HasPermission returns ErrInsufficientPermission unless the user holds every permission.
	HasPermission(ctx context.Context, user *limen.User, permissions access.Permissions) error
}

// Use returns the admin plugin API. Panics if the plugin is not registered.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginAdmin)
}

var _ API = (*adminPlugin)(nil)
