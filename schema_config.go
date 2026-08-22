package limen

// PluginSchemaConfig represents customization for a plugin schema
type PluginSchemaConfig struct {
	TableName        SchemaTableName        //  override table name
	Fields           map[SchemaField]string // Map of logical field name -> actual column name
	AdditionalFields AdditionalFieldsFunc
}

type PluginSchemaConfigOption func(*PluginSchemaConfig)

type SchemaConfig struct {
	// A function to return a map of global fields to be added to all schemas when writing a record. e.g:
	//  func(ctx *AdditionalFieldsContext) (map[string]any, error) {
	// 		return map[string]any{
	//  		"uuid": uuid.New().String(),
	//  		"created_at": time.Now(),
	//  		"updated_at": time.Now(),
	// 		 }, nil
	//	 }
	// this function will be called during the creation and the update of any schema record,
	AdditionalFields AdditionalFieldsFunc
	// IDGenerator generates IDs for all schemas
	IDGenerator IDGenerator
	// Account schema configuration
	Account *AccountSchema
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
	// Plugin schema customizations: PluginName -> SchemaName -> Config
	pluginSchemas map[PluginName]map[SchemaName]PluginSchemaConfig
	// Model transformers by logical schema name
	modelTransformers ModelTransformers
	// Resolved global public-ID configuration.
	publicID            *PublicIDConfig
	publicIDDisabledFor map[SchemaName]struct{}
}

type SchemaConfigOption func(*SchemaConfig)

// NewDefaultSchemaConfig creates a new SchemaConfig with default values.
func NewDefaultSchemaConfig(opts ...SchemaConfigOption) *SchemaConfig {
	config := &SchemaConfig{
		pluginSchemas:            make(map[PluginName]map[SchemaName]PluginSchemaConfig),
		coreSchemaCustomizations: make(map[SchemaName]PluginSchemaConfig),
		User:                     newDefaultUserSchema(nil),
		Verification:             newDefaultVerificationSchema(nil),
		Session:                  newDefaultSessionSchema(nil),
		RateLimit:                newDefaultRateLimitSchema(nil),
		Account:                  newDefaultAccountSchema(nil),
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

// MatchesIDColumnType checks if the value matches the ID column type
func (c *SchemaConfig) MatchesIDColumnType(value any) bool {
	idType := c.GetIDColumnType()
	switch idType {
	case ColumnTypeString, ColumnTypeText, ColumnTypeUUID:
		_, ok := value.(string)
		return ok
	case ColumnTypeInt, ColumnTypeInt32, ColumnTypeInt64:
		switch value.(type) {
		case int, int8, int16, int32, int64:
			return true
		case float64: // JSON numbers decode as float64 into `any`
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// NormalizeIDValue converts a JSON-decoded identifier back to the configured
// ID column type; other values pass through unchanged.
func (c *SchemaConfig) NormalizeIDValue(value any) any {
	v, ok := value.(float64)
	if !ok {
		return value
	}

	switch c.GetIDColumnType() {
	case ColumnTypeInt, ColumnTypeInt32, ColumnTypeInt64:
		return int64(v)
	default:
		return value
	}
}

func (c *SchemaConfig) getPublicIDConfig(schemaName SchemaName) (*PublicIDConfig, bool) {
	if c.publicID == nil || c.publicID.Disabled {
		return nil, false
	}
	if _, disabled := c.publicIDDisabledFor[schemaName]; disabled {
		return nil, false
	}
	return c.publicID, true
}

// getCoreSchemaCustomizationField returns the customized column name for a core schema field if set
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
