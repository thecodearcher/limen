package aegis

import "fmt"

// FieldResolver resolves logical field names to concrete column names
// using the discovered schema map.
type FieldResolver struct {
	schemas map[string]SchemaDefinition
}

// NewFieldResolver creates a new resolver from discovered schemas.
func NewFieldResolver(schemas map[string]SchemaDefinition) *FieldResolver {
	return &FieldResolver{schemas: schemas}
}

// ResolveField returns the concrete column name for a logical field within a schema.
// schemaName: the logical schema identifier (e.g. "users" or plugin schema name)
// logicalField: the logical field name as declared by the plugin/core schema
func (r *FieldResolver) ResolveField(schemaName, logicalField string) (string, error) {
	schema, ok := r.schemas[schemaName]
	if !ok {
		return "", fmt.Errorf("schema %s not found", schemaName)
	}

	for _, col := range schema.Columns {
		if col.LogicalField == logicalField {
			return col.Name, nil
		}
	}

	return "", fmt.Errorf("field %s not found in schema %s", logicalField, schemaName)
}

// MustResolveField returns the concrete column name or panics if missing.
func (r *FieldResolver) MustResolveField(schemaName, logicalField string) string {
	name, err := r.ResolveField(schemaName, logicalField)
	if err != nil {
		panic(err)
	}
	return name
}
