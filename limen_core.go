package limen

import (
	"context"
	"fmt"
	"maps"
	"net/http"
)

type LimenCore struct {
	config         *Config
	baseURL        string
	fullBaseURL    string // baseURL + HTTP.basePath
	db             DatabaseAdapter
	cacheStore     CacheAdapter
	DBAction       *DatabaseActionHelper
	Schema         *SchemaConfig
	SessionManager SessionManager
	cookies        *cookieManager
	schemaResolver *SchemaResolver
	plugins        map[PluginName]Plugin
	secret         []byte
}

func (c *LimenCore) getPublicIDConfig(schema Schema) (SchemaName, *PublicIDConfig, bool) {
	schemaName := schema.GetSchemaName()
	if schemaName == "" {
		return "", nil, false
	}

	config, ok := c.Schema.getPublicIDConfig(schemaName)
	if !ok {
		return schemaName, nil, false
	}
	return schemaName, config, true
}

func (c *LimenCore) EncodePublicID(schema Schema, model Model) (string, bool) {
	_, config, ok := c.getPublicIDConfig(schema)
	if !ok || model == nil {
		return "", false
	}

	column := schema.GetField(config.field)
	value, ok := model.Raw()[column].(string)
	if !ok || value == "" {
		return "", false
	}
	encoded := c.encodePublicIDValue(schema, value)
	return encoded, encoded != ""
}

func (c *LimenCore) IsPublicID(schema Schema, value string) bool {
	schemaName, config, ok := c.getPublicIDConfig(schema)
	if !ok {
		return false
	}
	return config.Matcher(schemaName, value)
}

func (c *LimenCore) DecodePublicID(schema Schema, value string) (string, error) {
	schemaName, config, ok := c.getPublicIDConfig(schema)
	if !ok {
		return "", fmt.Errorf("failed to get public ID config for schema %s", schema.GetSchemaName())
	}
	return config.Decoder(schemaName, value)
}

// GeneratePublicID calls the configured public-ID Generator for schema.
func (c *LimenCore) GeneratePublicID(ctx context.Context, schema Schema) (string, error) {
	schemaName, config, ok := c.getPublicIDConfig(schema)
	if !ok || config.Generator == nil {
		return "", fmt.Errorf("public ID generator is not configured for schema %s", schema.GetSchemaName())
	}
	return config.Generator(ctx, schemaName)
}

func (c *LimenCore) encodePublicIDValue(schema Schema, value string) string {
	schemaName, config, ok := c.getPublicIDConfig(schema)
	if !ok || value == "" {
		return ""
	}
	return config.Encoder(schemaName, value)
}

// PublicIDColumn returns the database column name for schema's public ID field,
// or "" when public IDs are not enabled.
func (c *LimenCore) PublicIDColumn(schema Schema) string {
	_, config, ok := c.getPublicIDConfig(schema)
	if !ok {
		return ""
	}
	return schema.GetField(config.field)
}

func (c *LimenCore) SerializeModel(schema Schema, model Model) map[string]any {
	serialized := maps.Clone(schema.Serialize(model))
	_, config, enabled := c.getPublicIDConfig(schema)

	if _, ok := serialized[schema.GetIDField()].(string); !ok {
		delete(serialized, schema.GetIDField())
	}

	if !enabled || config.DisableResponseTransform {
		return serialized
	}

	delete(serialized, schema.GetField(config.field))
	if encoded, ok := c.EncodePublicID(schema, model); ok {
		serialized[config.ResponseField] = encoded
	}
	return serialized
}

func (a *LimenCore) initializeSchemas(discoveredSchemas map[SchemaName]SchemaDefinition) error {
	if a.schemaResolver == nil {
		return fmt.Errorf("schema resolver must be instantiated before initializing schemas")
	}

	for schemaName, schema := range discoveredSchemas {
		schemaInfo := newSchemaInfo(schemaName, schema.TableName, a.schemaResolver)
		if err := schema.Schema.Initialize(schemaInfo); err != nil {
			return fmt.Errorf("failed to initialize schema instance for %s: %w", schemaName, err)
		}
	}
	return nil
}

// GetPlugin retrieves a plugin by its name from the plugin registry.
// Returns the plugin and true if found, or nil and false if not found.
func (c *LimenCore) GetPlugin(name PluginName) (Plugin, bool) {
	plugin, ok := c.plugins[name]
	return plugin, ok
}

// Cookies returns the shared CookieManager that plugins should use for
// all cookie operations. The returned manager inherits security attributes
// from the central cookie configuration.
func (c *LimenCore) Cookies() *cookieManager {
	return c.cookies
}

// CacheStore returns the global CacheAdapter instance.
// Plugins should use this as a fallback when no per-feature store is configured.
func (c *LimenCore) CacheStore() CacheAdapter {
	return c.cacheStore
}

// AtomicCacheStore returns the global AtomicCacheAdapter instance.
// Plugins should use this when they need atomic operations on the cache (e.g. increment/decrement).
func (c *LimenCore) AtomicCacheStore() AtomicCacheAdapter {
	return c.cacheStore.(AtomicCacheAdapter)
}

// CacheKeyPrefix returns the prefix used for all cache keys (sessions, rate limits).
func (c *LimenCore) CacheKeyPrefix() string {
	return c.config.CacheKeyPrefix
}

// Secret returns the base signing secret.
// Plugins that do not configure their own secret can use this for encryption/signing.
func (c *LimenCore) Secret() []byte {
	return c.secret
}

func (c *LimenCore) GetBaseURL() string {
	return c.baseURL
}

func (c *LimenCore) GetFullBaseURL() string {
	return c.fullBaseURL
}

func (c *LimenCore) GetBaseURLWithPluginPath(pluginName PluginName, pathToJoin string) string {
	plugin, ok := c.GetPlugin(pluginName)
	if !ok {
		return ""
	}

	pluginConfig := plugin.PluginHTTPConfig()
	normalizedBasePath := normalizePluginPath(c.config.HTTP.basePath, pluginConfig.BasePath, c.config.HTTP.overrides[string(pluginName)])
	return joinURL(c.baseURL, normalizedBasePath, pathToJoin)
}

// CreateSession creates a session for the auth result.
// This should be called instead of SessionManager.CreateSession so that plugins
// can pass options (e.g. remember_me) via SessionCreateOption.
func (c *LimenCore) CreateSession(ctx context.Context, r *http.Request, w http.ResponseWriter, auth *AuthenticationResult, opts ...SessionCreateOption) (*SessionResult, error) {
	createOpts := &SessionCreateOptions{ShortSession: false}
	if c.cookies.checkIsShortSession(r) {
		createOpts.ShortSession = true
	}
	for _, opt := range opts {
		opt(createOpts)
	}
	return c.SessionManager.CreateSession(ctx, r, auth, createOpts.ShortSession)
}

// RotateSession revokes the caller's current session (or all of the user's
// sessions when revokeAll is true) and issues a fresh session for the same
// user. The user is re-loaded from the database so the new session and
// response reflect any state changed earlier in the request.
func (c *LimenCore) RotateSession(r *http.Request, w http.ResponseWriter, session *ValidatedSession, revokeAll bool) (*AuthenticationResult, *SessionResult, error) {
	ctx := r.Context()

	if revokeAll {
		_ = c.SessionManager.RevokeAllSessions(ctx, session.User.ID)
	} else {
		_ = c.SessionManager.RevokeSession(ctx, session.Session.Token)
	}

	user, err := c.DBAction.FindUserByID(ctx, session.User.ID)
	if err != nil {
		return nil, nil, err
	}

	authResult := &AuthenticationResult{User: user}

	sessionResult, err := c.CreateSession(ctx, r, w, authResult)
	if err != nil {
		return nil, nil, err
	}

	return authResult, sessionResult, nil
}
