package limen

// SchemaInfo is a convenience struct that provides resolved schema
// information to schemas and wraps the SchemaResolver to make specific schema field lookups easier.
type SchemaInfo struct {
	schemaName SchemaName
	tableName  SchemaTableName
	resolver   *SchemaResolver
}

func newSchemaInfo(schemaName SchemaName, tableNames SchemaTableName, fieldResolver *SchemaResolver) *SchemaInfo {
	return &SchemaInfo{
		schemaName: schemaName,
		tableName:  tableNames,
		resolver:   fieldResolver,
	}
}

// GetField returns the resolved column name for a logical field.
func (m *SchemaInfo) GetField(logicalField SchemaField) string {
	return m.resolver.GetField(m.schemaName, logicalField)
}
