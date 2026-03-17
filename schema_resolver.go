package limen

type SchemaFieldMap map[SchemaName]map[SchemaField]string

// SchemaResolver resolves logical field names to concrete column names
// using the discovered schema map.
type SchemaResolver struct {
	tableNames map[SchemaName]SchemaTableName
	fields     SchemaFieldMap
}

// newFieldResolver creates a new resolver from discovered schemas.
func newFieldResolver(schemas SchemaDefinitionMap) *SchemaResolver {
	fieldSchemas := make(SchemaFieldMap)
	tableNames := make(map[SchemaName]SchemaTableName)
	for schemaName, schema := range schemas {
		fieldSchemas[schemaName] = make(map[SchemaField]string)
		tableNames[schemaName] = schema.TableName
		for _, col := range schema.Columns {
			fieldSchemas[schemaName][col.LogicalField] = col.Name
		}
	}

	return &SchemaResolver{fields: fieldSchemas, tableNames: tableNames}
}

func (r *SchemaResolver) GetTableName(schemaName SchemaName) SchemaTableName {
	return r.tableNames[schemaName]
}

func (r *SchemaResolver) GetFields(schemaName SchemaName) map[SchemaField]string {
	return r.fields[schemaName]
}

// GetField returns the concrete column name for a logical field within a schema.
func (r *SchemaResolver) GetField(schemaName SchemaName, logicalField SchemaField) string {
	return r.fields[schemaName][logicalField]
}
