package aegis

// SchemaExtension represents fields to add to an existing core schema
type SchemaExtension struct {
	Extends     string                 // Name of the core schema to extend (e.g., "users")
	Fields      []FieldDefinition      // Fields to add to the core schema
	Indexes     []IndexDefinition      // Indexes to add
	ForeignKeys []ForeignKeyDefinition // Foreign keys to add
}

// NewTableSchema represents a completely new table schema from a plugin
type NewTableSchema struct {
	TableName   TableName
	Fields      []FieldDefinition
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
}

// ToSchemaIntrospector converts a NewTableSchema to a SchemaIntrospector
func (n *NewTableSchema) ToSchemaIntrospector() SchemaIntrospector {
	return &newTableSchemaIntrospector{schema: n}
}

type newTableSchemaIntrospector struct {
	schema *NewTableSchema
}

func (n *newTableSchemaIntrospector) GetTableName() TableName {
	return n.schema.TableName
}

func (n *newTableSchemaIntrospector) GetFields() []FieldDefinition {
	return n.schema.Fields
}

func (n *newTableSchemaIntrospector) GetIndexes() []IndexDefinition {
	return n.schema.Indexes
}

func (n *newTableSchemaIntrospector) GetForeignKeys() []ForeignKeyDefinition {
	return n.schema.ForeignKeys
}

func (n *newTableSchemaIntrospector) GetExtends() string {
	return ""
}

// ToSchemaIntrospector converts a SchemaExtension to a SchemaIntrospector
func (s *SchemaExtension) ToSchemaIntrospector(schemaName string) SchemaIntrospector {
	return &schemaExtensionIntrospector{
		extension:  s,
		schemaName: schemaName,
	}
}

type schemaExtensionIntrospector struct {
	extension  *SchemaExtension
	schemaName string
}

func (s *schemaExtensionIntrospector) GetTableName() TableName {
	return TableName(s.schemaName)
}

func (s *schemaExtensionIntrospector) GetFields() []FieldDefinition {
	return s.extension.Fields
}

func (s *schemaExtensionIntrospector) GetIndexes() []IndexDefinition {
	return s.extension.Indexes
}

func (s *schemaExtensionIntrospector) GetForeignKeys() []ForeignKeyDefinition {
	return s.extension.ForeignKeys
}

func (s *schemaExtensionIntrospector) GetExtends() string {
	return s.extension.Extends
}
