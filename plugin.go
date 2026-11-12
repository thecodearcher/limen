package limen

import "context"

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	// Unique identifier for the plugin.
	Name() PluginName
	// Initialize initializes the plugin.
	Initialize(core *LimenCore) error
	// PluginHTTPConfig returns the configuration for the plugin's HTTP surface.
	PluginHTTPConfig() PluginHTTPConfig
	// RegisterRoutes registers routes for the plugin.
	RegisterRoutes(httpCore *LimenHTTPCore, routeBuilder *RouteBuilder)
}

// SchemaProvider is an optional interface that plugins can implement to
// contribute or modify database schemas.
type SchemaProvider interface {
	// GetSchemas returns all schemas provided by this plugin.
	// Returns a map of schema name to SchemaIntrospector.
	// Plugins can extend core schemas by setting Extends field, or create new tables.
	// If a plugin extends a core schema, it should return a schema with the same name
	// and set Extends to the core schema name (e.g., "users").
	GetSchemas(schema *SchemaConfig) []SchemaIntrospector
}

// SessionManagerProvider is an optional interface that plugins can implement
// to provide an alternative SessionManager. The core detects this during
// initialization and wires the session manager automatically.
type SessionManagerProvider interface {
	SessionManager() SessionManager
}

// SessionCreateGuard is an optional interface that plugins can implement
// to reject a session before it is created. An error from BeforeSessionCreate
// is returned by CreateSession.
type SessionCreateGuard interface {
	BeforeSessionCreate(ctx context.Context, auth *AuthenticationResult) error
}

// SessionExtension holds the fields a SessionResponseExtender adds to a session response.
type SessionExtension struct {
	// User fields are merged into the response's "user".
	User map[string]any
	// Session fields are merged into the response's "session".
	Session map[string]any
}

// SessionResponseExtender is an optional interface that plugins can implement
// to add fields to session responses. session is the session the response is
// for, or nil when it is unknown.
type SessionResponseExtender interface {
	ExtendSessionResponse(ctx context.Context, user *User, session *Session) SessionExtension
}

// PluginHTTPConfig is the configuration for the plugin's HTTP surface.
type PluginHTTPConfig struct {
	// The base path where the plugin's routes will be mounted.
	// This is relative to the Limen base path and can be overridden by the end user.
	BasePath string
	// Middleware to be applied to the plugin's routes.
	Middleware []Middleware
	// Hooks run before/after requests. PathMatcher, when set, restricts which paths trigger the hooks
	Hooks *Hooks
	// Specific rate limit rules to be applied to the plugin's routes.
	// These rules can be overridden by the end user.
	RateLimitRules []*RateLimitRule
}
