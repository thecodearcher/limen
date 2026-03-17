package limen

import (
	"context"
	"fmt"
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
	cookies        *CookieManager
	schemaResolver *SchemaResolver
	plugins        map[PluginName]Plugin
	secret         []byte
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
func (c *LimenCore) Cookies() *CookieManager {
	return c.cookies
}

// CacheStore returns the global CacheAdapter instance.
// Plugins should use this as a fallback when no per-feature store is configured.
func (c *LimenCore) CacheStore() CacheAdapter {
	return c.cacheStore
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
