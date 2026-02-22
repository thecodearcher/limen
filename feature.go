package aegis

// Feature is the interface that all features must implement.
type Feature interface {
	// Unique identifier for the feature.
	Name() FeatureName
	// Initialize initializes the feature.
	Initialize(core *AegisCore) error
	// PluginHTTPConfig returns the configuration for the plugin's HTTP surface.
	PluginHTTPConfig() PluginHTTPConfig
	// RegisterRoutes registers routes for the plugin.
	RegisterRoutes(httpCore *AegisHTTPCore, routeBuilder *RouteBuilder)
	// GetSchemas returns all schemas provided by this feature.
	// Returns a map of schema name to SchemaIntrospector.
	// Plugins can extend core schemas by setting Extends field, or create new tables.
	// If a plugin extends a core schema, it should return a schema with the same name
	// and set Extends to the core schema name (e.g., "users").
	GetSchemas(schema *SchemaConfig) []SchemaIntrospector
}

// PluginHTTPConfig is the configuration for the plugin's HTTP surface.
type PluginHTTPConfig struct {
	// The base path where the plugin's routes will be mounted.
	// This is relative to the Aegis base path and can be overridden by the end user.
	BasePath string
	// Middleware to be applied to the plugin's routes.
	Middleware []Middleware
	// Hooks run before/after requests. PathMatcher, when set, restricts which paths trigger the hooks
	Hooks *Hooks
	// Specific rate limit rules to be applied to the plugin's routes.
	// These rules can be overridden by the end user.
	RateLimitRules []*RateLimitRule
}
