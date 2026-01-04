package aegis

// CoreSchemaName represents a valid core schema name that can be extended by plugins
type CoreSchemaName string

const (
	// CoreSchemaUsers is the name of the users core schema
	CoreSchemaUsers CoreSchemaName = "users"
	// CoreSchemaSessions is the name of the sessions core schema
	CoreSchemaSessions CoreSchemaName = "sessions"
	// CoreSchemaVerifications is the name of the verifications core schema
	CoreSchemaVerifications CoreSchemaName = "verifications"
	// CoreSchemaRateLimits is the name of the rate_limits core schema
	CoreSchemaRateLimits CoreSchemaName = "rate_limits"
)

// IsValidCoreSchema checks if a string is a valid core schema name
func IsValidCoreSchema(name string) bool {
	switch CoreSchemaName(name) {
	case CoreSchemaUsers, CoreSchemaSessions, CoreSchemaVerifications, CoreSchemaRateLimits:
		return true
	default:
		return false
	}
}

// ColumnType represents a Go type for a database column
type ColumnType string

const (
	// ColumnTypeUUID represents the uuid string type
	ColumnTypeUUID ColumnType = "uuid"
	// ColumnTypeString represents the string type
	ColumnTypeString ColumnType = "string"
	// ColumnTypeInt represents the int type
	ColumnTypeInt ColumnType = "int"
	// ColumnTypeInt32 represents the int32 type
	ColumnTypeInt32 ColumnType = "int32"
	// ColumnTypeInt64 represents the int64 type
	ColumnTypeInt64 ColumnType = "int64"
	// ColumnTypeBool represents the bool type
	ColumnTypeBool ColumnType = "bool"
	// ColumnTypeTime represents the time.Time type
	ColumnTypeTime ColumnType = "time.Time"
	// ColumnTypeAny represents the any type
	ColumnTypeAny ColumnType = "any"
	// ColumnTypeMapStringAny represents the map[string]any type
	ColumnTypeMapStringAny ColumnType = "map[string]any"
)

// ForeignKeyAction represents a SQL foreign key action
type ForeignKeyAction string

const (
	// FKActionCascade represents CASCADE action
	FKActionCascade ForeignKeyAction = "CASCADE"
	// FKActionSetNull represents SET NULL action
	FKActionSetNull ForeignKeyAction = "SET NULL"
	// FKActionRestrict represents RESTRICT action
	FKActionRestrict ForeignKeyAction = "RESTRICT"
	// FKActionNoAction represents NO ACTION
	FKActionNoAction ForeignKeyAction = "NO ACTION"
	// FKActionSetDefault represents SET DEFAULT action
	FKActionSetDefault ForeignKeyAction = "SET DEFAULT"
)

// SchemaIntrospector provides introspection capabilities for a schema
type SchemaIntrospector interface {
	// GetTableName returns the table name for this schema
	GetTableName() SchemaTableName
	// GetColumns returns all column definitions for this schema
	GetColumns() []ColumnDefinition
	// GetIndexes returns all index definitions for this schema
	GetIndexes() []IndexDefinition
	// GetForeignKeys returns all foreign key definitions for this schema
	GetForeignKeys() []ForeignKeyDefinition
	// GetExtends returns the name of the core schema this extends, or nil if none
	GetExtends() *CoreSchemaName
	// GetSchemaName returns the name of the logical schema name
	GetSchemaName() string
	// GetSchema returns the schema instance
	// The schema is of type Schema[M] where M is the model type
	// we don't want to use generics here because we want to support multiple model types
	GetSchema() Schema
}

// ColumnDefinition represents a single field/column in a schema
type ColumnDefinition struct {
	Name         string            // Actual database column name (respects custom field mappings)
	LogicalField string            // Logical field identifier: SchemaField constant value for core schemas, field name for plugin schemas
	Type         ColumnType        // Go type (use ColumnType constants or NewColumnType for custom types)
	IsNullable   bool              // Whether the field can be NULL
	IsPrimaryKey bool              // Whether this is a primary key
	DefaultValue string            // Default value (empty if none)
	Tags         map[string]string // JSON, gorm, sql tags
}

// IndexDefinition represents a database index
type IndexDefinition struct {
	Name    string   // Index name
	Columns []string // Column names in the index
	Unique  bool     // Whether this is a unique index
}

// ForeignKeyDefinition represents a foreign key relationship
type ForeignKeyDefinition struct {
	Name             string           // Foreign key constraint name
	Column           string           // Local column name (uses schema.Fields)
	ReferencedSchema SchemaTableName  // Schema name (e.g., "users" or plugin schema name) - symbolic reference
	ReferencedField  SchemaField      // Field name (e.g., "id" or field constant) - symbolic reference
	OnDelete         ForeignKeyAction // ON DELETE action (use FKAction constants)
	OnUpdate         ForeignKeyAction // ON UPDATE action (use FKAction constants)
}
