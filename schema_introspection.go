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

// AllCoreSchemaNames returns all valid core schema names
func AllCoreSchemaNames() []CoreSchemaName {
	return []CoreSchemaName{
		CoreSchemaUsers,
		CoreSchemaSessions,
		CoreSchemaVerifications,
		CoreSchemaRateLimits,
	}
}

// ColumnType represents a Go type for a database column
type ColumnType string

const (
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
	// ColumnTypeTimePtr represents the *time.Time type
	ColumnTypeTimePtr ColumnType = "*time.Time"
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
	GetTableName() TableName
	// GetColumns returns all column definitions for this schema
	GetColumns() []ColumnDefinition
	// GetIndexes returns all index definitions for this schema
	GetIndexes() []IndexDefinition
	// GetForeignKeys returns all foreign key definitions for this schema
	GetForeignKeys() []ForeignKeyDefinition
	// GetExtends returns the name of the core schema this extends, or nil if none
	GetExtends() *CoreSchemaName
}

// baseIntrospector is a generic base implementation of SchemaIntrospector
type baseIntrospector[T any] struct {
	tableName      TableName
	getColumns     func(T) []ColumnDefinition
	getIndexes     func(T) []IndexDefinition
	getForeignKeys func(T) []ForeignKeyDefinition
	getExtends     func(T) *CoreSchemaName
	schema         T
}

func (b *baseIntrospector[T]) GetTableName() TableName {
	return b.tableName
}

func (b *baseIntrospector[T]) GetColumns() []ColumnDefinition {
	return b.getColumns(b.schema)
}

func (b *baseIntrospector[T]) GetIndexes() []IndexDefinition {
	return b.getIndexes(b.schema)
}

func (b *baseIntrospector[T]) GetForeignKeys() []ForeignKeyDefinition {
	return b.getForeignKeys(b.schema)
}

func (b *baseIntrospector[T]) GetExtends() *CoreSchemaName {
	return b.getExtends(b.schema)
}

// NewIntrospector creates a new generic introspector for a schema type
func NewIntrospector[T any](
	schema T,
	tableName TableName,
	getColumns func(T) []ColumnDefinition,
	getIndexes func(T) []IndexDefinition,
	getForeignKeys func(T) []ForeignKeyDefinition,
	getExtends func(T) *CoreSchemaName,
) SchemaIntrospector {
	return &baseIntrospector[T]{
		tableName:      tableName,
		getColumns:     getColumns,
		getIndexes:     getIndexes,
		getForeignKeys: getForeignKeys,
		getExtends:     getExtends,
		schema:         schema,
	}
}

// Introspect converts a SchemaIntrospector to a TableDefinition
func Introspect(introspector SchemaIntrospector) SchemaDefinition {
	return SchemaDefinition{
		TableName:   introspector.GetTableName(),
		Columns:     introspector.GetColumns(),
		Indexes:     introspector.GetIndexes(),
		ForeignKeys: introspector.GetForeignKeys(),
		Extends:     introspector.GetExtends(),
	}
}

// SchemaDefinition represents a complete schema definition
type SchemaDefinition struct {
	TableName   TableName
	Columns     []ColumnDefinition
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
	Extends     *CoreSchemaName // If extending a core schema (e.g., CoreSchemaUsers), nil for new tables
	PluginName  string          // Name of the plugin that owns this schema, empty for core schemas
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
	Name                 string           // Foreign key constraint name
	Column               string           // Local column name (uses schema.Fields)
	ReferencedSchema     string           // Schema name (e.g., "users" or plugin schema name) - symbolic reference
	ReferencedField      string           // Field name (e.g., "id" or field constant) - symbolic reference
	ReferencedTableName  TableName        // Resolved table name (populated during schema discovery)
	ReferencedColumnName string           // Resolved column name (populated during schema discovery)
	OnDelete             ForeignKeyAction // ON DELETE action (use FKAction constants)
	OnUpdate             ForeignKeyAction // ON UPDATE action (use FKAction constants)
}
