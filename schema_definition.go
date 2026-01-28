package aegis

import (
	"fmt"
)

// SchemaDefinition represents a complete schema definition.
type SchemaDefinition struct {
	TableName   SchemaTableName
	Columns     []ColumnDefinition
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
	SchemaName  SchemaName // Name of the schema
	Extends     SchemaName // If extending a core schema (e.g., CoreSchemaUsers), nil for new tables
	PluginName  string     // Name of the plugin that owns this schema, empty for core schemas
	Schema      Schema     `json:"-"` // Schema instance (excluded from JSON serialization for CLI)
}

// For extensions, it uses the SchemaName as a temporary table name.
// The actual table name will be resolved during schema discovery from the core schema.
func (d *SchemaDefinition) GetTableName() SchemaTableName {
	if d.Extends != "" {
		return SchemaTableName(string(d.Extends))
	}

	if d.TableName != "" {
		return d.TableName
	}

	panic("SchemaDefinition: either Extends or TableName must be set")
}

func (d *SchemaDefinition) GetColumns() []ColumnDefinition {
	return d.Columns
}

func (d *SchemaDefinition) GetIndexes() []IndexDefinition {
	return d.Indexes
}

func (d *SchemaDefinition) GetForeignKeys() []ForeignKeyDefinition {
	return d.ForeignKeys
}

func (d *SchemaDefinition) GetExtends() SchemaName {
	return d.Extends
}

func (d *SchemaDefinition) GetSchemaName() SchemaName {
	return d.SchemaName
}

func (d *SchemaDefinition) GetSchema() Schema {
	return d.Schema
}

// NewSchemaDefinitionForTable creates a new SchemaDefinition for a new table
func NewSchemaDefinitionForTable(schemaName SchemaName, tableName SchemaTableName, schema Schema, opts ...SchemaDefinitionOption) *SchemaDefinition {
	def := &SchemaDefinition{
		TableName:   tableName,
		Columns:     []ColumnDefinition{},
		Indexes:     []IndexDefinition{},
		ForeignKeys: []ForeignKeyDefinition{},
		SchemaName:  schemaName,
		Schema:      schema,
	}

	if isCoreSchema(schema) {
		panic(fmt.Sprintf("Schema type %T is either a core schema or embeds a core schema and cannot be used as a new table schema", schema))
	}

	for _, opt := range opts {
		opt(def)
	}
	return def
}

// NewSchemaDefinitionForExtension creates a new SchemaDefinition for extending a core schema
func NewSchemaDefinitionForExtension(schemaName SchemaName, modifiedSchema Schema, opts ...SchemaDefinitionOption) *SchemaDefinition {
	def := &SchemaDefinition{
		Extends:     schemaName,
		Columns:     []ColumnDefinition{},
		Indexes:     []IndexDefinition{},
		ForeignKeys: []ForeignKeyDefinition{},
		SchemaName:  schemaName,
		Schema:      modifiedSchema,
	}
	for _, opt := range opts {
		opt(def)
	}
	return def
}

type SchemaDefinitionOption func(*SchemaDefinition)

func WithSchemaIDField(config *SchemaConfig) SchemaDefinitionOption {
	idType := config.GetIDColumnType()

	return func(d *SchemaDefinition) {
		d.Columns = append(d.Columns, ColumnDefinition{
			Name:         string(SchemaIDField),
			LogicalField: SchemaIDField,
			Type:         idType,
			IsNullable:   false,
			IsPrimaryKey: true,
		})
	}
}

// WithSchemaField adds a field to the schema.
// If the logical field name is not provided, it will be set to the name parameter.
func WithSchemaField(name string, columnType ColumnType, opts ...ColumnDefinitionOption) SchemaDefinitionOption {
	return func(d *SchemaDefinition) {
		field := ColumnDefinition{
			Name:         name,
			LogicalField: SchemaField(name),
			Type:         columnType,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags:         make(map[string]string),
		}

		for _, opt := range opts {
			opt(&field)
		}

		d.Columns = append(d.Columns, field)
	}
}

// WithSchemaIndex adds an index to the schema
func WithSchemaIndex(name string, columns []SchemaField) SchemaDefinitionOption {
	return func(d *SchemaDefinition) {
		d.Indexes = append(d.Indexes, IndexDefinition{
			Name:    name,
			Columns: columns,
			Unique:  false,
		})
	}
}

// WithSchemaIndex adds an index to the schema
func WithSchemaUniqueIndex(name string, columns []SchemaField) SchemaDefinitionOption {
	return func(d *SchemaDefinition) {
		d.Indexes = append(d.Indexes, IndexDefinition{
			Name:    name,
			Columns: columns,
			Unique:  true,
		})
	}
}

// WithSchemaForeignKey adds a foreign key to the schema
func WithSchemaForeignKey(foreignKey ForeignKeyDefinition) SchemaDefinitionOption {
	return func(d *SchemaDefinition) {
		d.ForeignKeys = append(d.ForeignKeys, foreignKey)
	}
}

type ColumnDefinitionOption func(*ColumnDefinition)

// WithLogicalField sets the logical field name (different from column name)
func WithLogicalField(logicalField SchemaField) ColumnDefinitionOption {
	return func(f *ColumnDefinition) {
		f.LogicalField = logicalField
	}
}

// WithNullable sets whether the field can be NULL
func WithNullable(nullable bool) ColumnDefinitionOption {
	return func(f *ColumnDefinition) {
		f.IsNullable = nullable
	}
}

// WithPrimaryKey sets whether this is a primary key
func WithPrimaryKey(pk bool) ColumnDefinitionOption {
	return func(f *ColumnDefinition) {
		f.IsPrimaryKey = pk
	}
}

// WithDefaultValue sets the default value for the field
func WithDefaultValue(defaultValue string) ColumnDefinitionOption {
	return func(f *ColumnDefinition) {
		f.DefaultValue = defaultValue
	}
}

// WithTags sets tags for the field (JSON, gorm, sql, etc.)
func WithTags(tags map[string]string) ColumnDefinitionOption {
	return func(f *ColumnDefinition) {
		f.Tags = tags
	}
}
