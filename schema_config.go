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
	// User schema configuration
	User *UserSchema
	// Verification schema configuration
	Verification *VerificationSchema
	// Session schema configuration
	Session *SessionSchema
	// Rate limit schema configuration
	RateLimit *RateLimitSchema
	// Plugin schema customizations: FeatureName -> SchemaName -> Config
	PluginSchemas map[FeatureName]map[string]PluginSchemaConfig
}

type SchemaConfigOption func(*SchemaConfig)

// NewDefaultSchemaConfig creates a new SchemaConfig with default values.
func NewDefaultSchemaConfig(opts ...SchemaConfigOption) *SchemaConfig {
	config := &SchemaConfig{
		User:          NewDefaultUserSchema(),
		Verification:  NewDefaultVerificationSchema(),
		Session:       NewDefaultSessionSchema(),
		RateLimit:     NewDefaultRateLimitSchema(),
		PluginSchemas: make(map[FeatureName]map[string]PluginSchemaConfig),
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// WithSchemaAdditionalFields sets the global additional fields function
func WithSchemaAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.AdditionalFields = fn
	}
}

// WithSchemaUser sets the user schema configuration
func WithSchemaUser(opts ...UserSchemaOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.User = NewDefaultUserSchema(opts...)
	}
}

// WithSchemaVerification sets the verification schema configuration
func WithSchemaVerification(opts ...VerificationSchemaOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Verification = NewDefaultVerificationSchema(opts...)
	}
}

// WithSchemaSession sets the session schema configuration
func WithSchemaSession(opts ...SessionSchemaOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Session = NewDefaultSessionSchema(opts...)
	}
}

// WithSchemaRateLimit sets the rate limit schema configuration
func WithSchemaRateLimit(opts ...RateLimitSchemaOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.RateLimit = NewDefaultRateLimitSchema(opts...)
	}
}

// PluginSchemaConfig represents customization for a plugin schema
type PluginSchemaConfig struct {
	TableName *TableName        // Optional: override table name
	Fields    map[string]string // Map of logical field name -> actual column name
}

type PluginSchemaConfigOption func(*PluginSchemaConfig)

// WithPluginTableName sets the table name for a plugin schema
func WithPluginTableName(tableName TableName) PluginSchemaConfigOption {
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
