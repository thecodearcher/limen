package aegis

// SchemaIntrospector provides introspection capabilities for a schema
type SchemaIntrospector interface {
	// GetTableName returns the table name for this schema
	GetTableName() TableName
	// GetFields returns all field definitions for this schema
	GetFields() []FieldDefinition
	// GetIndexes returns all index definitions for this schema
	GetIndexes() []IndexDefinition
	// GetForeignKeys returns all foreign key definitions for this schema
	GetForeignKeys() []ForeignKeyDefinition
	// GetExtends returns the name of the core schema this extends, or empty string if none
	GetExtends() string
}

// Introspect converts a SchemaIntrospector to a SchemaDefinition
func Introspect(introspector SchemaIntrospector) SchemaDefinition {
	return SchemaDefinition{
		TableName:   introspector.GetTableName(),
		Fields:      introspector.GetFields(),
		Indexes:     introspector.GetIndexes(),
		ForeignKeys: introspector.GetForeignKeys(),
		Extends:     introspector.GetExtends(),
	}
}

// SchemaDefinition represents a complete schema definition
type SchemaDefinition struct {
	TableName   TableName
	Fields      []FieldDefinition
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
	Extends     string // If extending a core schema (e.g., "users")
	PluginName  string // Name of the plugin that owns this schema, empty for core schemas
}

// FieldDefinition represents a single field/column in a schema
type FieldDefinition struct {
	Name         string            // Actual database column name (respects custom field mappings)
	Type         string            // Go type (string, int, time.Time, *time.Time, etc.)
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
	Name             string // Foreign key constraint name
	Column           string // Local column name
	ReferencedTable  string // Referenced table name
	ReferencedColumn string // Referenced column name
	OnDelete         string // ON DELETE action (CASCADE, SET NULL, etc.)
	OnUpdate         string // ON UPDATE action
}
