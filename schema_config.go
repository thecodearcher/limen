package aegis

// PluginSchemaConfig represents customization for a plugin schema
type PluginSchemaConfig struct {
	TableName SchemaTableName        //  override table name
	Fields    map[SchemaField]string // Map of logical field name -> actual column name
}

type PluginSchemaConfigOption func(*PluginSchemaConfig)

type SchemaConfig struct {
	// A function to return a map of global fields to be added to all schemas when creating a record. e.g:
	//  func(ctx context.Context) map[string]any {
	// 		return map[string]any{
	//  		"uuid": uuid.New().String(),
	//  		"created_at": time.Now(),
	//  		"updated_at": time.Now(),
	// 		 }
	//	 }
	// this function will be called during the creation of any schema record.
	// You can also set fields on supported schemas itself.
	AdditionalFields AdditionalFieldsFunc
	// IDGenerator generates IDs for all schemas
	IDGenerator IDGenerator
	// User schema configuration
	User *UserSchema
	// Verification schema configuration
	Verification *VerificationSchema
	// Session schema configuration
	Session *SessionSchema
	// Rate limit schema configuration
	RateLimit *RateLimitSchema
	// Core schema customizations
	coreSchemaCustomizations map[SchemaName]PluginSchemaConfig
	// Plugin schema customizations: FeatureName -> SchemaName -> Config
	pluginSchemas map[FeatureName]map[SchemaName]PluginSchemaConfig
}

type SchemaConfigOption func(*SchemaConfig)

// NewDefaultSchemaConfig creates a new SchemaConfig with default values.
func NewDefaultSchemaConfig(opts ...SchemaConfigOption) *SchemaConfig {
	config := &SchemaConfig{
		pluginSchemas:            make(map[FeatureName]map[SchemaName]PluginSchemaConfig),
		coreSchemaCustomizations: make(map[SchemaName]PluginSchemaConfig),
		User:                     newDefaultUserSchema(nil),
		Verification:             newDefaultVerificationSchema(nil),
		Session:                  newDefaultSessionSchema(nil),
		RateLimit:                newDefaultRateLimitSchema(nil),
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// GetIDColumnType returns the ColumnType for ID fields based on the configured ID generator
// Returns ColumnTypeInt64 (for auto-increment) if no generator is configured
func (c *SchemaConfig) GetIDColumnType() ColumnType {
	if c != nil && c.IDGenerator != nil {
		return c.IDGenerator.GetColumnType()
	}
	return ColumnTypeInt64
}

func (c *SchemaConfig) getCoreSchemaCustomizationField(schemaName SchemaName, field SchemaField) string {
	exists, ok := c.coreSchemaCustomizations[schemaName]
	if !ok || exists.Fields == nil {
		return ""
	}
	return exists.Fields[field]
}

func (c *SchemaConfig) setCoreSchemaField(schemaName SchemaName, field SchemaField, value string) {
	if exists, ok := c.coreSchemaCustomizations[schemaName]; ok {
		if exists.Fields == nil {
			exists.Fields = make(map[SchemaField]string)
		}
		exists.Fields[field] = value
		c.coreSchemaCustomizations[schemaName] = exists
		return
	}

	c.coreSchemaCustomizations[schemaName] = PluginSchemaConfig{
		Fields: map[SchemaField]string{
			field: value,
		},
	}
}

func (c *SchemaConfig) setCoreSchemaTableName(schemaName SchemaName, tableName SchemaTableName) {
	if exists, ok := c.coreSchemaCustomizations[schemaName]; ok {
		exists.TableName = tableName
		c.coreSchemaCustomizations[schemaName] = exists
		return
	}

	c.coreSchemaCustomizations[schemaName] = PluginSchemaConfig{
		TableName: tableName,
		Fields:    make(map[SchemaField]string),
	}
}

// WithSchemaAdditionalFields sets the global additional fields function
func WithSchemaAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.AdditionalFields = fn
	}
}

// WithSchemaIDGenerator sets the global ID generator
func WithSchemaIDGenerator(generator IDGenerator) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.IDGenerator = generator
	}
}

// WithSchemaUser sets the user schema configuration
func WithSchemaUser(opts ...SchemaConfigUserOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.User = newDefaultUserSchema(c, opts...)
	}
}

// WithSchemaVerification sets the verification schema configuration
func WithSchemaVerification(opts ...SchemaConfigVerificationOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Verification = newDefaultVerificationSchema(c, opts...)
	}
}

// WithSchemaSession sets the session schema configuration
func WithSchemaSession(opts ...SchemaConfigSessionOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Session = newDefaultSessionSchema(c, opts...)
	}
}

// WithSchemaRateLimit sets the rate limit schema configuration
func WithSchemaRateLimit(opts ...SchemaConfigRateLimitOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.RateLimit = newDefaultRateLimitSchema(c, opts...)
	}
}

// WithPluginTableName sets the table name for a plugin schema
func WithPluginTableName(tableName SchemaTableName) PluginSchemaConfigOption {
	return func(c *PluginSchemaConfig) {
		c.TableName = tableName
	}
}

// WithPluginFieldName sets a field name mapping for a plugin schema
func WithPluginFieldName(logicalField SchemaField, columnName string) PluginSchemaConfigOption {
	return func(c *PluginSchemaConfig) {
		if c.Fields == nil {
			c.Fields = make(map[SchemaField]string)
		}
		c.Fields[logicalField] = columnName
	}
}

// WithPluginSchema sets the configuration for a plugin schema
func WithPluginSchema(featureName FeatureName, schemaName SchemaName, opts ...PluginSchemaConfigOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		if c.pluginSchemas[featureName] == nil {
			c.pluginSchemas[featureName] = make(map[SchemaName]PluginSchemaConfig)
		}
		config := PluginSchemaConfig{
			Fields: make(map[SchemaField]string),
		}
		for _, opt := range opts {
			opt(&config)
		}
		c.pluginSchemas[featureName][schemaName] = config
	}
}
