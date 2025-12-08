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

			// Check if this schema extends a core schema
			if def.Extends != "" {
				// Merge the extended fields into the core schema
				coreSchema, exists := schemas[def.Extends]
				if !exists {
					return nil, fmt.Errorf("plugin %s extends unknown schema: %s", feature.Name(), def.Extends)
				}

				// Merge fields (plugin fields take precedence for same field names)
				mergedFields := make(map[string]FieldDefinition)
				for _, field := range coreSchema.Fields {
					mergedFields[field.Name] = field
				}
				for _, field := range def.Fields {
					// Check for conflicts
					if existing, exists := mergedFields[field.Name]; exists {
						// Allow override if it's from a plugin extending the schema
						if existing.Name == field.Name && existing.Type != field.Type {
							return nil, fmt.Errorf("plugin %s conflicts with existing field %s in schema %s: type mismatch (%s vs %s)",
								feature.Name(), field.Name, def.Extends, existing.Type, field.Type)
						}
					}
					mergedFields[field.Name] = field
				}

				// Convert back to slice
				mergedFieldsSlice := make([]FieldDefinition, 0, len(mergedFields))
				for _, field := range mergedFields {
					mergedFieldsSlice = append(mergedFieldsSlice, field)
				}

				// Merge indexes
				mergedIndexes := append(coreSchema.Indexes, def.Indexes...)

				// Merge foreign keys
				mergedForeignKeys := append(coreSchema.ForeignKeys, def.ForeignKeys...)

				// Update the core schema with merged data
				coreSchema.Fields = mergedFieldsSlice
				coreSchema.Indexes = mergedIndexes
				coreSchema.ForeignKeys = mergedForeignKeys
				schemas[def.Extends] = coreSchema
			} else {
				// New table from plugin
				if _, exists := schemas[schemaName]; exists {
					return nil, fmt.Errorf("plugin %s defines schema %s which already exists", feature.Name(), schemaName)
				}
				schemas[schemaName] = def
			}
		}
	}

	return schemas, nil
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
