package aegis

// PluginSchema represents a schema definition for a plugin.
// It can either extend an existing core schema or define a new table.
type PluginSchema struct {
	TableName   *SchemaTableName // For new tables (nil if extending a core schema)
	Extends     *CoreSchemaName  // For extending core schemas (nil if new table)
	Fields      []ColumnDefinition
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
	SchemaName  string
}

// NewPluginSchemaForTable creates a new PluginSchema for a new table
func NewPluginSchemaForTable(schemaName SchemaName, tableName SchemaTableName, opts ...PluginSchemaOption) *PluginSchema {
	p := &PluginSchema{
		TableName:   &tableName,
		Extends:     nil,
		Fields:      []ColumnDefinition{},
		Indexes:     []IndexDefinition{},
		ForeignKeys: []ForeignKeyDefinition{},
		SchemaName:  string(schemaName),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type PluginSchemaOption func(*PluginSchema)

func WithPluginSchemaField(name string, columnType ColumnType, opts ...FieldOption) PluginSchemaOption {
	return func(p *PluginSchema) {
		p.Fields = append(p.Fields, WithField(name, columnType, opts...))
	}
}

func WithPluginSchemaIndex(index IndexDefinition) PluginSchemaOption {
	return func(p *PluginSchema) {
		p.Indexes = append(p.Indexes, index)
	}
}

func WithPluginSchemaForeignKey(foreignKey ForeignKeyDefinition) PluginSchemaOption {
	return func(p *PluginSchema) {
		p.ForeignKeys = append(p.ForeignKeys, foreignKey)
	}
}

// NewPluginSchemaForExtension creates a new PluginSchema for extending a core schema
func NewPluginSchemaForExtension(schemaName CoreSchemaName, opts ...PluginSchemaOption) *PluginSchema {
	p := &PluginSchema{
		TableName:   nil,
		Extends:     &schemaName,
		Fields:      []ColumnDefinition{},
		Indexes:     []IndexDefinition{},
		ForeignKeys: []ForeignKeyDefinition{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
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

// ToSchemaIntrospector converts a PluginSchema to a SchemaIntrospector.
// For extensions (p.Extends != nil), it uses the CoreSchemaName as a temporary table name.
// The actual table name will be resolved during schema discovery from the core schema.
// For new tables (p.TableName != nil), it uses the provided table name directly.
func (p *PluginSchema) ToSchemaIntrospector() SchemaIntrospector {
	var tableName SchemaTableName
	if p.Extends != nil {
		// For extensions, use CoreSchemaName as temporary table name
		// Actual table name will be resolved during discovery
		tableName = SchemaTableName(string(*p.Extends))
	} else if p.TableName != nil {
		// For new tables, use the provided table name
		tableName = *p.TableName
	} else {
		panic("PluginSchema: either Extends or TableName must be set")
	}

	return p.toSchemaIntrospectorWithTableName(tableName)
}

// toSchemaIntrospectorWithTableName is the internal implementation
func (p *PluginSchema) toSchemaIntrospectorWithTableName(tableName SchemaTableName) SchemaIntrospector {

	return NewIntrospector(
		p,
		tableName,
		p.SchemaName,
		func(s *PluginSchema) []ColumnDefinition {
			return s.Fields
		},
		func(s *PluginSchema) []IndexDefinition {
			return s.Indexes
		},
		func(s *PluginSchema) []ForeignKeyDefinition {
			return s.ForeignKeys
		},
		func(s *PluginSchema) *CoreSchemaName {
			return s.Extends
		},
	)
}
