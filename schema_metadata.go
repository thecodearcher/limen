package aegis

// PluginSchemaMetadata provides resolved schema information to plugins.
// It contains the actual table name, field mappings, and resolver after user customizations.
type PluginSchemaMetadata struct {
	// SchemaName is the resolved schema name.
	// For extensions: the core schema name being extended (e.g., "users").
	// For new tables: the plugin's schema name.
	SchemaName string
	// TableName is the actual table name after user customizations.
	TableName SchemaTableName
	// FieldResolver provides access to resolve logical field names to column names.
	FieldResolver *FieldResolver
	// Fields maps logical field names declared by this plugin to their resolved column names.
	// This only includes fields declared by the plugin itself.
	// This provides O(1) access for plugin-declared fields.
	Fields map[string]string
}

// GetField returns the resolved column name for a logical field.
// It first checks Fields (plugin's own fields), then falls back to FieldResolver.
func (m *PluginSchemaMetadata) GetField(logicalField string) (string, error) {
	// First check if this is a field declared by the plugin
	if col, ok := m.Fields[logicalField]; ok {
		return col, nil
	}
	// Fallback to FieldResolver for fields not declared by this plugin (e.g., core fields)
	return m.FieldResolver.ResolveField(m.SchemaName, logicalField)
}

// MustGetField returns the resolved column name or panics if not found.
func (m *PluginSchemaMetadata) MustGetField(logicalField string) string {
	field, err := m.GetField(logicalField)
	if err != nil {
		panic(err)
	}
	return field
}
