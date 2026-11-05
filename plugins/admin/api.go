package admin

import (
	"context"
	"net/http"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type API interface {
	// ListRoles returns each configured role's expanded permissions. Requires user:set-role.
	ListRoles(ctx context.Context, actor *limen.User) (map[string]access.Permissions, error)
	// GetUserRoles returns the user's assigned roles, or the default roles when none are assigned.
	GetUserRoles(ctx context.Context, user *limen.User) ([]access.Role, error)
	// GetPermissions returns the union of the user's role permissions.
	GetPermissions(ctx context.Context, user *limen.User) (access.Permissions, error)
	// HasPermission returns ErrInsufficientPermission unless the user holds every permission.
	HasPermission(ctx context.Context, user *limen.User, permissions access.Permissions) error
	// SetRoles replaces the user's assigned roles. Requires user:set-role.
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
	// SetEmail changes a user's email. Requires user:set-email.
	SetEmail(ctx context.Context, actor *limen.User, userID any, req *SetEmailRequest) (*limen.User, error)
	// SetUserPassword replaces a user's password. Requires user:set-password.
	SetUserPassword(ctx context.Context, actor *limen.User, userID any, req *SetUserPasswordRequest) error
	// BanUser bans a user and revokes their sessions. Requires user:ban.
	BanUser(ctx context.Context, actor *limen.User, userID any, req *BanUserRequest) (*limen.User, error)
	// UnbanUser lifts a user's ban. Requires user:ban.
	UnbanUser(ctx context.Context, actor *limen.User, userID any) (*limen.User, error)
	// ListUserSessions returns the user's sessions. Requires session:list.
	ListUserSessions(ctx context.Context, actor *limen.User, userID any) ([]UserSession, error)
	// RevokeSession revokes a session by its token. Requires session:revoke.
	RevokeSession(ctx context.Context, actor *limen.User, token string) error
	// RevokeUserSessions revokes all of the user's sessions. Requires session:revoke.
	RevokeUserSessions(ctx context.Context, actor *limen.User, userID any) error
	// ImpersonateUser creates a session as the user, recorded as impersonated by the session's user.
	// Requires user:impersonate, and user:impersonate-admins when the user has the admin role.
	ImpersonateUser(ctx context.Context, r *http.Request, session *limen.ValidatedSession, userID any) (*limen.User, *limen.SessionResult, error)
	// StopImpersonating revokes the impersonation session and returns the impersonator.
	StopImpersonating(ctx context.Context, session *limen.ValidatedSession) (*limen.User, error)
	// DeleteUser deletes a user and their sessions and accounts. Requires user:delete.
	DeleteUser(ctx context.Context, actor *limen.User, userID any) error
}

// Use returns the admin plugin API. Panics if the plugin is not registered.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginAdmin)
}

var (
	_ API                      = (*adminPlugin)(nil)
	_ limen.SessionCreateGuard = (*adminPlugin)(nil)
)
