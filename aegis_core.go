package aegis

import "fmt"

type AegisCore struct {
	config         *Config
	baseURL        string
	fullBaseURL    string // baseURL + HTTP.basePath
	db             DatabaseAdapter
	DBAction       *DatabaseActionHelper
	Schema         *SchemaConfig
	SessionManager SessionManager
	cookies        *CookieManager
	schemaResolver *SchemaResolver
	features       map[FeatureName]Feature
}

func (a *AegisCore) initializeSchemas(discoveredSchemas map[SchemaName]SchemaDefinition) error {
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

// GetFeature retrieves a feature by its name from the plugin registry.
// Returns the feature and true if found, or nil and false if not found.
func (c *AegisCore) GetFeature(name FeatureName) (Feature, bool) {
	feature, ok := c.features[name]
	return feature, ok
}

// Cookies returns the shared CookieManager that plugins should use for
// all cookie operations. The returned manager inherits security attributes
// from the central cookie configuration.
func (c *AegisCore) Cookies() *CookieManager {
	return c.cookies
}

func (c *AegisCore) GetBaseURL() string {
	return c.baseURL
}

func (c *AegisCore) GetFullBaseURL() string {
	return c.fullBaseURL
}

func (c *AegisCore) GetBaseURLWithPluginPath(pluginName FeatureName, pathToJoin string) string {
	feature, ok := c.GetFeature(pluginName)
	if !ok {
		return ""
	}

	featureConfig := feature.PluginHTTPConfig()
	normalizedBasePath := normalizePluginPath(c.config.HTTP.basePath, featureConfig.BasePath, c.config.HTTP.overrides[string(pluginName)])
	return joinURL(c.baseURL, normalizedBasePath, pathToJoin)
}
