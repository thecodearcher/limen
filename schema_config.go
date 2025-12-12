package aegis

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
	coreSchemaCustomizations map[CoreSchemaName]PluginSchemaConfig
	// Plugin schema customizations: FeatureName -> SchemaName -> Config
	PluginSchemas map[FeatureName]map[string]PluginSchemaConfig
}

type SchemaConfigOption func(*SchemaConfig)

// NewDefaultSchemaConfig creates a new SchemaConfig with default values.
func NewDefaultSchemaConfig(opts ...SchemaConfigOption) *SchemaConfig {
	config := &SchemaConfig{
		PluginSchemas:            make(map[FeatureName]map[string]PluginSchemaConfig),
		coreSchemaCustomizations: make(map[CoreSchemaName]PluginSchemaConfig),
	}

	for _, opt := range opts {
		opt(config)
	}

	// Set defaults if not provided
	if config.Verification == nil {
		config.Verification = newDefaultVerificationSchema(config)
	}
	if config.Session == nil {
		config.Session = newDefaultSessionSchema(config)
	}
	if config.RateLimit == nil {
		config.RateLimit = newDefaultRateLimitSchema(config)
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

func (c *SchemaConfig) getCoreSchemaCustomizationField(schemaName CoreSchemaName, field string) string {
	exists, ok := c.coreSchemaCustomizations[schemaName]
	if !ok || exists.Fields == nil {
		return ""
	}
	return exists.Fields[field]
}

func (c *SchemaConfig) setCoreSchemaField(schemaName CoreSchemaName, field string, value string) {
	if exists, ok := c.coreSchemaCustomizations[schemaName]; ok {
		if exists.Fields == nil {
			exists.Fields = make(map[string]string)
		}
		exists.Fields[field] = value
		c.coreSchemaCustomizations[schemaName] = exists
		return
	}

	c.coreSchemaCustomizations[schemaName] = PluginSchemaConfig{
		Fields: map[string]string{
			field: value,
		},
	}
}

func (c *SchemaConfig) setCoreSchemaTableName(schemaName CoreSchemaName, tableName SchemaTableName) {
	if exists, ok := c.coreSchemaCustomizations[schemaName]; ok {
		exists.TableName = &tableName
		c.coreSchemaCustomizations[schemaName] = exists
		return
	}

	c.coreSchemaCustomizations[schemaName] = PluginSchemaConfig{
		TableName: &tableName,
		Fields:    make(map[string]string),
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

// PluginSchemaConfig represents customization for a plugin schema
type PluginSchemaConfig struct {
	TableName *SchemaTableName  // Optional: override table name
	Fields    map[string]string // Map of logical field name -> actual column name
}

type PluginSchemaConfigOption func(*PluginSchemaConfig)

// WithPluginTableName sets the table name for a plugin schema
func WithPluginTableName(tableName SchemaTableName) PluginSchemaConfigOption {
	return func(c *PluginSchemaConfig) {
		c.TableName = &tableName
	}
}

// WithPluginFieldName sets a field name mapping for a plugin schema
func WithPluginFieldName(logicalField, columnName string) PluginSchemaConfigOption {
	return func(c *PluginSchemaConfig) {
		if c.Fields == nil {
			c.Fields = make(map[string]string)
		}
		c.Fields[logicalField] = columnName
	}
}

// WithPluginSchema sets the configuration for a plugin schema
func WithPluginSchema(featureName FeatureName, schemaName string, opts ...PluginSchemaConfigOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		if c.PluginSchemas == nil {
			c.PluginSchemas = make(map[FeatureName]map[string]PluginSchemaConfig)
		}
		if c.PluginSchemas[featureName] == nil {
			c.PluginSchemas[featureName] = make(map[string]PluginSchemaConfig)
		}
		config := PluginSchemaConfig{
			Fields: make(map[string]string),
		}
		for _, opt := range opts {
			opt(&config)
		}
		c.PluginSchemas[featureName][schemaName] = config
	}
}
