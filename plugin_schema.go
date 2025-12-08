package aegis

// PluginSchema represents a schema definition for a plugin.
// It can either extend an existing core schema or define a new table.
type PluginSchema struct {
	TableName   *TableName      // For new tables (nil if extending a core schema)
	Extends     *CoreSchemaName // For extending core schemas (nil if new table)
	Fields      []ColumnDefinition
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
}

// NewPluginSchemaForTable creates a new PluginSchema for a new table
func NewPluginSchemaForTable(tableName TableName) *PluginSchema {
	return &PluginSchema{
		TableName:   &tableName,
		Extends:     nil,
		Fields:      []ColumnDefinition{},
		Indexes:     []IndexDefinition{},
		ForeignKeys: []ForeignKeyDefinition{},
	}
}

// NewPluginSchemaForExtension creates a new PluginSchema for extending a core schema
func NewPluginSchemaForExtension(schemaName CoreSchemaName) *PluginSchema {
	return &PluginSchema{
		TableName:   nil,
		Extends:     &schemaName,
		Fields:      []ColumnDefinition{},
		Indexes:     []IndexDefinition{},
		ForeignKeys: []ForeignKeyDefinition{},
	}
}

// ToSchemaIntrospector converts a PluginSchema to a SchemaIntrospector.
// For extensions (p.Extends != nil), it uses the CoreSchemaName as a temporary table name.
// The actual table name will be resolved during schema discovery from the core schema.
// For new tables (p.TableName != nil), it uses the provided table name directly.
func (p *PluginSchema) ToSchemaIntrospector() SchemaIntrospector {
	var tableName TableName
	if p.Extends != nil {
		// For extensions, use CoreSchemaName as temporary table name
		// Actual table name will be resolved during discovery
		tableName = TableName(string(*p.Extends))
	} else if p.TableName != nil {
		// For new tables, use the provided table name
		tableName = *p.TableName
	} else {
		panic("PluginSchema: either Extends or TableName must be set")
	}

	return p.toSchemaIntrospectorWithTableName(tableName)
}

// toSchemaIntrospectorWithTableName is the internal implementation
func (p *PluginSchema) toSchemaIntrospectorWithTableName(tableName TableName) SchemaIntrospector {

	return NewIntrospector(
		p,
		tableName,
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
