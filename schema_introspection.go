package aegis

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
	GetSchema() Schema
}

// ColumnDefinition represents a single field/column in a schema
type ColumnDefinition struct {
	Name         string            // Actual database column name (respects custom field mappings)
	LogicalField string            // Logical field identifier: SchemaField constant value for core schemas, field name for plugin schemas
	Type         ColumnType        // Go type (use ColumnType constants)
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
	ReferencedSchema SchemaTableName  // Schema name of the referenced schema
	ReferencedField  SchemaField      // Field name of the referenced field
	OnDelete         ForeignKeyAction // ON DELETE action (use FKAction constants)
	OnUpdate         ForeignKeyAction // ON UPDATE action (use FKAction constants)
}
