package aegis

// SchemaDefinition represents a complete schema definition.
type SchemaDefinition struct {
	TableName   *SchemaTableName // Pointer to table name (nil for extensions before discovery)
	Columns     []ColumnDefinition
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
	SchemaName  string          // Name of the schema
	Extends     *CoreSchemaName // If extending a core schema (e.g., CoreSchemaUsers), nil for new tables
	PluginName  string          // Name of the plugin that owns this schema, empty for core schemas
	Schema      Schema          `json:"-"` // Schema instance (excluded from JSON serialization for CLI)
}

// GetTableName implements SchemaIntrospector.
// For extensions (Extends != nil), it uses the CoreSchemaName as a temporary table name.
// The actual table name will be resolved during schema discovery from the core schema.
// For new tables (TableName != nil), it uses the provided table name directly.
func (d *SchemaDefinition) GetTableName() SchemaTableName {
	if d.Extends != nil {
		// For extensions, use CoreSchemaName as temporary table name
		// Actual table name will be resolved during discovery
		return SchemaTableName(string(*d.Extends))
	}
	if d.TableName != nil {
		// For new tables, use the provided table name
		return *d.TableName
	}
	panic("SchemaDefinition: either Extends or TableName must be set")
}

// GetColumns implements SchemaIntrospector.
func (d *SchemaDefinition) GetColumns() []ColumnDefinition {
	return d.Columns
}

// GetIndexes implements SchemaIntrospector.
func (d *SchemaDefinition) GetIndexes() []IndexDefinition {
	return d.Indexes
}

// GetForeignKeys implements SchemaIntrospector.
func (d *SchemaDefinition) GetForeignKeys() []ForeignKeyDefinition {
	return d.ForeignKeys
}

// GetExtends implements SchemaIntrospector.
func (d *SchemaDefinition) GetExtends() *CoreSchemaName {
	return d.Extends
}

// GetSchemaName implements SchemaIntrospector.
func (d *SchemaDefinition) GetSchemaName() string {
	return d.SchemaName
}

// GetSchema implements SchemaIntrospector.
func (d *SchemaDefinition) GetSchema() Schema {
	return d.Schema
}

// NewSchemaDefinitionForTable creates a new SchemaDefinition for a new table
func NewSchemaDefinitionForTable(schemaName SchemaName, tableName SchemaTableName, schema Schema, opts ...SchemaDefinitionOption) *SchemaDefinition {
	def := &SchemaDefinition{
		TableName:   &tableName,
		Extends:     nil,
		Columns:     []ColumnDefinition{},
		Indexes:     []IndexDefinition{},
		ForeignKeys: []ForeignKeyDefinition{},
		SchemaName:  string(schemaName),
		Schema:      schema,
	}
	for _, opt := range opts {
		opt(def)
	}
	return def
}

// NewSchemaDefinitionForExtension creates a new SchemaDefinition for extending a core schema
func NewSchemaDefinitionForExtension(schemaName CoreSchemaName, schema Schema, opts ...SchemaDefinitionOption) *SchemaDefinition {
	def := &SchemaDefinition{
		TableName:   nil,
		Extends:     &schemaName,
		Columns:     []ColumnDefinition{},
		Indexes:     []IndexDefinition{},
		ForeignKeys: []ForeignKeyDefinition{},
		SchemaName:  string(schemaName),
		Schema:      schema,
	}
	for _, opt := range opts {
		opt(def)
	}
	return def
}

// SchemaDefinitionOption configures a SchemaDefinition when building
type SchemaDefinitionOption func(*SchemaDefinition)

// WithSchemaField creates an option to add a field to the schema
func WithSchemaField(name string, columnType ColumnType, opts ...FieldOption) SchemaDefinitionOption {
	return func(d *SchemaDefinition) {
		d.Columns = append(d.Columns, WithField(name, columnType, opts...))
	}
}

// WithSchemaIndex creates an option to add an index
func WithSchemaIndex(index IndexDefinition) SchemaDefinitionOption {
	return func(d *SchemaDefinition) {
		d.Indexes = append(d.Indexes, index)
	}
}

// WithSchemaForeignKey creates an option to add a foreign key
func WithSchemaForeignKey(foreignKey ForeignKeyDefinition) SchemaDefinitionOption {
	return func(d *SchemaDefinition) {
		d.ForeignKeys = append(d.ForeignKeys, foreignKey)
	}
}

// FieldOption configures a ColumnDefinition when adding a field
type FieldOption func(*ColumnDefinition)

// WithLogicalField sets the logical field name (different from column name)
// If not provided, LogicalField defaults to the Name parameter
func WithLogicalField(logicalField string) FieldOption {
	return func(f *ColumnDefinition) {
		f.LogicalField = logicalField
	}
}

// WithNullable sets whether the field can be NULL
func WithNullable(nullable bool) FieldOption {
	return func(f *ColumnDefinition) {
		f.IsNullable = nullable
	}
}

// WithPrimaryKey sets whether this is a primary key
func WithPrimaryKey(pk bool) FieldOption {
	return func(f *ColumnDefinition) {
		f.IsPrimaryKey = pk
	}
}

// WithDefaultValue sets the default value for the field
func WithDefaultValue(defaultValue string) FieldOption {
	return func(f *ColumnDefinition) {
		f.DefaultValue = defaultValue
	}
}

// WithTags sets tags for the field (JSON, gorm, sql, etc.)
func WithTags(tags map[string]string) FieldOption {
	return func(f *ColumnDefinition) {
		f.Tags = tags
	}
}

// WithField adds a field to the schema using builder pattern
// LogicalField is automatically set to name if not explicitly provided via WithLogicalField option
func WithField(name string, columnType ColumnType, opts ...FieldOption) ColumnDefinition {
	field := ColumnDefinition{
		Name:         name,
		LogicalField: name,
		Type:         columnType,
		IsNullable:   false,
		IsPrimaryKey: false,
		Tags:         make(map[string]string),
	}

	for _, opt := range opts {
		opt(&field)
	}

	return field
}
