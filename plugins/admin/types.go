package admin

import (
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type UserSerializerFunc func(user *limen.User) map[string]any

type UserSession struct {
	ID             string      `json:"id"`
	CreatedAt      time.Time   `json:"created_at"`
	ExpiresAt      time.Time   `json:"expires_at"`
	LastAccess     time.Time   `json:"last_access,omitzero"`
	IPAddress      string      `json:"ip_address,omitempty"`
	UserAgent      string      `json:"user_agent,omitempty"`
	ImpersonatedBy *limen.User `json:"-"`
}

type userSessionResponse struct {
	UserSession
	ImpersonatedBy map[string]any `json:"impersonated_by,omitempty"`
}

type UserWithRoles struct {
	*limen.User
	Roles []access.Role
}

type ListUsersQuery struct {
	Search  string            `json:"search"`
	Filters map[string]string `json:"filter"`
}

type CreateUserRequest struct {
	Email            string         `json:"email"`
	FirstName        *string        `json:"first_name,omitempty"`
	LastName         *string        `json:"last_name,omitempty"`
	EmailVerified    bool           `json:"email_verified"`
	AdditionalFields map[string]any `json:"-"`
}

// SetEmailRequest changes a user's email. EmailVerified is ignored when SendVerification is set.
type SetEmailRequest struct {
	Email            string `json:"email"`
	EmailVerified    bool   `json:"email_verified"`
	SendVerification bool   `json:"send_verification"`
}

type SetUserPasswordRequest struct {
	Password       string `json:"password"`
	RevokeSessions bool   `json:"revoke_sessions"`
}

type UpdateUserRequest struct {
	FirstName        *string        `json:"first_name,omitempty"`
	LastName         *string        `json:"last_name,omitempty"`
	EmailVerified    *bool          `json:"email_verified,omitempty"`
	AdditionalFields map[string]any `json:"-"`
}

type config struct {
	accessControl        *access.AccessControl
	roles                []access.Role
	defaultRoles         []string
	adminRole            string
	adminUserIDs         []any
	userFilters          map[string]UserFilterFunc
	sortableUserFields   []limen.SchemaField
	searchColumns        []string
	userFields           []limen.SchemaField
	userSerializer       UserSerializerFunc
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

// WithUserFilter registers a filter clients can use as filter[name]=value.
// A filter with a built-in name replaces the built-in.
func WithUserFilter(name string, fn UserFilterFunc) ConfigOption {
	return func(c *config) {
		if c.userFilters == nil {
			c.userFilters = make(map[string]UserFilterFunc)
		}
		c.userFilters[name] = fn
	}
}

// WithSortableUserFields allows sort_by to name these fields on top of created_at and email.
func WithSortableUserFields(fields ...limen.SchemaField) ConfigOption {
	return func(c *config) {
		c.sortableUserFields = fields
	}
}

// WithUserSearchColumns adds columns that search matches on top of email, first name and last name.
func WithUserSearchColumns(columns ...string) ConfigOption {
	return func(c *config) {
		c.searchColumns = columns
	}
}

// WithUserFields limits the users in admin responses to the given fields.
// Never exposes more than the core user serializer does.
func WithUserFields(fields ...limen.SchemaField) ConfigOption {
	return func(c *config) {
		c.userFields = fields
	}
}

// WithUserSerializer builds the users in admin responses. It takes precedence over WithUserFields.
func WithUserSerializer(serializer UserSerializerFunc) ConfigOption {
	return func(c *config) {
		c.userSerializer = serializer
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
