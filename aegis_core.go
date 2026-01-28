package aegis

type AegisCore struct {
	db             DatabaseAdapter
	DBAction       *DatabaseActionHelper
	Schema         *SchemaConfig
	SessionManager *SessionManager
	schemaResolver *SchemaResolver
	features       map[FeatureName]Feature
}

// GetFeature retrieves a feature by its name from the plugin registry.
// Returns the feature and true if found, or nil and false if not found.
func (c *AegisCore) GetFeature(name FeatureName) (Feature, bool) {
	feature, ok := c.features[name]
	return feature, ok
}

// GetCredentialPasswordFeature retrieves the credential-password feature if available.
// Returns the CredentialPasswordFeature and true if found, or nil and false if not found.
func (c *AegisCore) GetCredentialPasswordFeature() CredentialPasswordFeature {
	feature, ok := c.GetFeature(FeatureCredentialPassword)
	if !ok {
		return nil
	}
	return feature.(CredentialPasswordFeature)
}
