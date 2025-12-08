package aegis

import (
	"fmt"
)

// DiscoverAllSchemas discovers all schemas from core and all registered features.
// It merges plugin-extended fields into core schemas and returns a complete schema map.
func (c *AegisCore) DiscoverAllSchemas(features []Feature) (map[string]SchemaDefinition, error) {
	schemas := make(map[string]SchemaDefinition)

	// First, collect all core schemas
	coreSchemas := map[string]SchemaIntrospector{
		"users":         c.Schema.User.Introspect(),
		"verifications": c.Schema.Verification.Introspect(),
		"sessions":      c.Schema.Session.Introspect(),
		"rate_limits":   c.Schema.RateLimit.Introspect(),
	}
	for name, introspector := range coreSchemas {
		schemas[name] = Introspect(introspector)
	}

	// Then, collect schemas from all features
	for _, feature := range features {
		featureSchemas := feature.GetSchemas()
		for schemaName, introspector := range featureSchemas {
			def := Introspect(introspector)
			def.PluginName = string(feature.Name())

			// Apply plugin schema customizations if they exist
			if schemaConfigs, exists := c.Schema.PluginSchemas[feature.Name()]; exists {
				if schemaConfig, exists := schemaConfigs[schemaName]; exists {
					// Override table name if provided
					if schemaConfig.TableName != nil {
						def.TableName = *schemaConfig.TableName
					}
					// Override field names if provided
					if len(schemaConfig.Fields) > 0 {
						for i := range def.Columns {
							col := &def.Columns[i]
							if newName, exists := schemaConfig.Fields[col.LogicalField]; exists {
								col.Name = newName
							}
						}
					}
				}
			}

			// Check if this schema extends a core schema
			if def.Extends != nil {
				extendsName := string(*def.Extends)
				// Validate that the extended schema is a valid core schema
				if !IsValidCoreSchema(extendsName) {
					return nil, fmt.Errorf("plugin %s extends invalid core schema: %s", feature.Name(), extendsName)
				}
				// Merge the extended fields into the core schema
				coreSchema, exists := schemas[extendsName]
				if !exists {
					return nil, fmt.Errorf("plugin %s extends unknown schema: %s", feature.Name(), extendsName)
				}

				// Merge fields (plugin fields take precedence for same field names)
				mergedFields := make(map[string]ColumnDefinition)
				for _, field := range coreSchema.Columns {
					mergedFields[field.Name] = field
				}
				for _, field := range def.Columns {
					existingCoreField, exists := mergedFields[field.Name]
					if !exists {
						mergedFields[field.Name] = field
						continue
					}

					if existingCoreField.Name == field.Name && existingCoreField.Type != field.Type {
						return nil, fmt.Errorf("plugin %s conflicts with existing field %s in schema %s: type mismatch (%s vs %s)",
							feature.Name(), field.Name, extendsName, existingCoreField.Type, field.Type)
					}
				}

				// Convert back to slice
				mergedFieldsSlice := make([]ColumnDefinition, 0, len(mergedFields))
				for _, field := range mergedFields {
					mergedFieldsSlice = append(mergedFieldsSlice, field)
				}

				// Merge indexes
				mergedIndexes := append(coreSchema.Indexes, def.Indexes...)

				// Merge foreign keys
				mergedForeignKeys := append(coreSchema.ForeignKeys, def.ForeignKeys...)

				// Update the core schema with merged data
				coreSchema.Columns = mergedFieldsSlice
				coreSchema.Indexes = mergedIndexes
				coreSchema.ForeignKeys = mergedForeignKeys
				schemas[extendsName] = coreSchema
			} else {
				// New table from plugin
				if _, exists := schemas[schemaName]; exists {
					return nil, fmt.Errorf("plugin %s defines schema %s which already exists", feature.Name(), schemaName)
				}
				schemas[schemaName] = def
			}
		}
	}

	if err := resolveForeignKeyReferences(schemas); err != nil {
		return nil, err
	}

	return schemas, nil
}

func resolveForeignKeyReferences(schemas map[string]SchemaDefinition) error {
	for schemaName, schema := range schemas {
		for i := range schema.ForeignKeys {
			fk := &schema.ForeignKeys[i]

			if fk.ReferencedTableName != "" && fk.ReferencedColumnName != "" {
				continue
			}

			referencedSchema, exists := schemas[fk.ReferencedSchema]
			if !exists {
				return fmt.Errorf("schema %s has foreign key referencing unknown schema: %s", schemaName, fk.ReferencedSchema)
			}

			var referencedColumn *ColumnDefinition
			for j := range referencedSchema.Columns {
				col := &referencedSchema.Columns[j]
				if col.LogicalField == fk.ReferencedField {
					referencedColumn = col
					break
				}
			}

			if referencedColumn == nil {
				return fmt.Errorf("schema %s has foreign key referencing unknown field %s in schema %s", schemaName, fk.ReferencedField, fk.ReferencedSchema)
			}

			fk.ReferencedTableName = referencedSchema.TableName
			fk.ReferencedColumnName = referencedColumn.Name
		}

		schemas[schemaName] = schema
	}

	return nil
}

// DiscoverAllSchemasFromConfig discovers all schemas from a Config struct.
// This is the main entry point for schema discovery.
func DiscoverAllSchemasFromConfig(config *Config) (map[string]SchemaDefinition, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	core := &AegisCore{
		DB:     config.Database,
		Schema: *config.Schema,
	}

	return core.DiscoverAllSchemas(config.Features)
}
