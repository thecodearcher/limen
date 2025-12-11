package aegis

import (
	"fmt"
)

// DiscoverAllSchemas discovers all schemas from core and all registered features.
// It merges plugin-extended fields into core schemas and returns a complete schema map.
func (a *AegisCore) DiscoverAllSchemas(features []Feature) (map[string]SchemaDefinition, error) {
	schemas := a.collectCoreSchemas()
	schemas = a.applyCoreSchemaCustomizations(schemas, a.Schema)

	for _, feature := range features {
		if err := a.processFeatureSchemas(feature, schemas); err != nil {
			return nil, err
		}
	}

	for schemaName, schema := range schemas {
		if err := validateSchemaFields(schema, schemaName, schema.PluginName); err != nil {
			return nil, err
		}
	}

	if err := resolveForeignKeyReferences(schemas); err != nil {
		return nil, err
	}

	return schemas, nil
}

// collectCoreSchemas collects all core schemas and returns them as a map
func (a *AegisCore) collectCoreSchemas() map[string]SchemaDefinition {
	return map[string]SchemaDefinition{
		string(CoreSchemaUsers):         Introspect(a.Schema.User.Introspect(a.Schema)),
		string(CoreSchemaVerifications): Introspect(a.Schema.Verification.Introspect(a.Schema)),
		string(CoreSchemaSessions):      Introspect(a.Schema.Session.Introspect()),
		string(CoreSchemaRateLimits):    Introspect(a.Schema.RateLimit.Introspect()),
	}
}

func (a *AegisCore) applyCoreSchemaCustomizations(schemas map[string]SchemaDefinition, config *SchemaConfig) map[string]SchemaDefinition {
	modifiedSchemas := make(map[string]SchemaDefinition)
	for schemaName, schema := range schemas {
		customConfig, exists := config.coreSchemaCustomizations[CoreSchemaName(schemaName)]
		if !exists {
			modifiedSchemas[schemaName] = schema
			continue
		}
		if customConfig.TableName != nil {
			schema.TableName = *customConfig.TableName
		}

		for i, col := range schema.Columns {
			if newName, exists := customConfig.Fields[col.LogicalField]; exists {
				col.Name = newName
				schema.Columns[i] = col
			}
		}
		modifiedSchemas[schemaName] = schema
	}
	return modifiedSchemas
}

// processFeatureSchemas processes all schemas from a feature and updates the schemas map
func (a *AegisCore) processFeatureSchemas(feature Feature, schemas map[string]SchemaDefinition) error {
	featureSchemas := feature.GetSchemas(a.Schema)
	for _, introspector := range featureSchemas {
		def := Introspect(introspector)
		def.PluginName = string(feature.Name())

		if err := applyPluginCustomizations(&def, feature.Name(), def.SchemaName, a.Schema); err != nil {
			return err
		}

		if def.Extends != nil {
			if err := a.mergeSchemaExtension(feature, &def, schemas); err != nil {
				return err
			}
			continue
		}

		if err := a.mergeSchemaTable(feature, &def, def.SchemaName, schemas); err != nil {
			return err
		}
	}

	return nil
}

// applyPluginCustomizations applies user customizations to a plugin schema definition
func applyPluginCustomizations(def *SchemaDefinition, featureName FeatureName, schemaName string, config *SchemaConfig) error {
	schemaConfigs, exists := config.PluginSchemas[featureName]
	if !exists {
		return nil
	}
	schemaConfig, exists := schemaConfigs[schemaName]
	if !exists || (schemaConfig.TableName == nil && len(schemaConfig.Fields) == 0) {
		return nil
	}

	if schemaConfig.TableName != nil {
		def.TableName = *schemaConfig.TableName
	}

	for i := range def.Columns {
		col := &def.Columns[i]
		if newName, exists := schemaConfig.Fields[col.LogicalField]; exists {
			col.Name = newName
		}
	}

	return nil
}

func (a *AegisCore) mergeSchemaTable(feature Feature, def *SchemaDefinition, schemaName string, schemas map[string]SchemaDefinition) error {
	if _, exists := schemas[schemaName]; exists {
		return fmt.Errorf("plugin %s defines schema %s which already exists", feature.Name(), schemaName)
	}

	for _, schema := range schemas {
		if schema.TableName != def.TableName {
			continue
		}

		pluginName := schema.PluginName
		if schema.PluginName == "" {
			pluginName = "core"
		}
		return fmt.Errorf("plugin (%s) defines schema %s which conflicts with schema (%s) which has table \"%s\"",
			feature.Name(), schemaName, pluginName, string(schema.TableName),
		)
	}
	schemas[schemaName] = *def
	return nil
}

// mergeSchemaExtension merges a plugin schema extension into a core schema
func (a *AegisCore) mergeSchemaExtension(feature Feature, def *SchemaDefinition, schemas map[string]SchemaDefinition) error {
	extendsName := string(*def.Extends)

	if !IsValidCoreSchema(extendsName) {
		return fmt.Errorf("plugin %s extends invalid core schema: %s", feature.Name(), extendsName)
	}

	coreSchema, exists := schemas[extendsName]
	if !exists {
		return fmt.Errorf("plugin %s extends unknown schema: %s", feature.Name(), extendsName)
	}

	coreSchema.Columns = append(coreSchema.Columns, def.Columns...)
	coreSchema.Indexes = append(coreSchema.Indexes, def.Indexes...)
	coreSchema.ForeignKeys = append(coreSchema.ForeignKeys, def.ForeignKeys...)
	schemas[extendsName] = coreSchema
	return nil
}

func resolveForeignKeyReferences(schemas map[string]SchemaDefinition) error {
	for schemaName, schema := range schemas {
		for i := range schema.ForeignKeys {
			fk := &schema.ForeignKeys[i]

			if fk.ReferencedTableName != "" && fk.ReferencedColumnName != "" {
				continue
			}

			referencedSchema, exists := schemas[string(fk.ReferencedSchema)]
			if !exists {
				return fmt.Errorf("schema %s has foreign key referencing unknown schema: %s", schemaName, fk.ReferencedSchema)
			}

			referencedColumn := findColumnByLogicalField(referencedSchema.Columns, SchemaField(fk.ReferencedField))
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

// validateSchemaFields validates that a schema definition has no internal conflicts
// It checks for duplicate logicalField and name values within the schema
func validateSchemaFields(schema SchemaDefinition, schemaName string, ownerName string) error {
	if ownerName == "" {
		ownerName = "core"
	}
	logicalFields := make(map[string]string)
	names := make(map[string]string)

	for _, col := range schema.Columns {
		if col.LogicalField == "" || col.Name == "" {
			return fmt.Errorf("schema %s (owner: %s) has column with empty logicalField or name", schemaName, ownerName)
		}

		if existingNameField, exists := logicalFields[col.LogicalField]; exists {
			return fmt.Errorf(
				"schema %s (owner: %s) has duplicate logicalField %s at field %s and %s",
				schemaName, ownerName, col.LogicalField, existingNameField, col.Name,
			)
		}

		if existingLogicalField, exists := names[col.Name]; exists {
			return fmt.Errorf(
				"schema %s (owner: %s) has duplicate name %s at field %s and %s",
				schemaName, ownerName, col.Name, existingLogicalField, col.LogicalField,
			)
		}

		logicalFields[col.LogicalField] = col.Name
		names[col.Name] = col.LogicalField
	}

	return nil
}

func findColumnByLogicalField(columns []ColumnDefinition, logicalField SchemaField) *ColumnDefinition {
	for _, col := range columns {
		if col.LogicalField == string(logicalField) {
			return &col
		}
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
		Schema: config.Schema,
	}

	return core.DiscoverAllSchemas(config.Features)
}

// buildPluginSchemaMetadata builds schema metadata for a feature's schemas.
// It returns a map where the key is the schema name that should be used to look up metadata.
func (a *AegisCore) buildPluginSchemaMetadata(feature Feature, discoveredSchemas map[string]SchemaDefinition) (map[string]*PluginSchemaMetadata, error) {
	metadata := make(map[string]*PluginSchemaMetadata)
	featureSchemas := feature.GetSchemas(a.Schema)

	for _, introspector := range featureSchemas {
		originalDef := Introspect(introspector)

		schemaName := originalDef.SchemaName
		if originalDef.Extends != nil {
			schemaName = string(*originalDef.Extends)
		}

		meta, err := a.buildSchemaMetadata(originalDef, discoveredSchemas)
		if err != nil {
			return nil, fmt.Errorf("failed to build schema metadata for %s: %w", schemaName, err)
		}

		metadata[schemaName] = meta

		if err := introspector.GetSchema().Initialize(a, meta); err != nil {
			return nil, fmt.Errorf("failed to initialize schema instance for %s: %w", schemaName, err)
		}
	}

	return metadata, nil
}

func (a *AegisCore) buildCoreSchemasMetadata(discoveredSchemas map[string]SchemaDefinition) (map[string]*PluginSchemaMetadata, error) {
	metadata := make(map[string]*PluginSchemaMetadata)
	schemas := []SchemaIntrospector{
		a.Schema.User.Introspect(a.Schema),
		a.Schema.Verification.Introspect(a.Schema),
		a.Schema.Session.Introspect(),
		a.Schema.RateLimit.Introspect(),
	}

	for _, introspector := range schemas {
		schema := Introspect(introspector)

		meta, err := a.buildSchemaMetadata(schema, discoveredSchemas)
		if err != nil {
			return nil, fmt.Errorf("failed to build schema fields for %s: %w", schema.SchemaName, err)
		}

		metadata[schema.SchemaName] = meta
		if err := introspector.GetSchema().Initialize(a, meta); err != nil {
			return nil, fmt.Errorf("failed to initialize schema instance for %s: %w", schema.SchemaName, err)
		}
	}
	return metadata, nil
}

func (a *AegisCore) buildSchemaMetadata(schema SchemaDefinition, discoveredSchemas map[string]SchemaDefinition) (*PluginSchemaMetadata, error) {
	resolvedSchema, exists := discoveredSchemas[schema.SchemaName]
	if !exists {
		return nil, fmt.Errorf("schema %s not found", schema.SchemaName)
	}

	// Build field mappings for fields declared by this plugin
	// We only include fields that were in the original plugin definition
	fields := make(map[string]string)
	for _, col := range schema.Columns {
		for _, resolvedCol := range resolvedSchema.Columns {
			if resolvedCol.LogicalField == col.LogicalField {
				fields[col.LogicalField] = resolvedCol.Name
				break
			}
		}
	}

	return &PluginSchemaMetadata{
		SchemaName:    schema.SchemaName,
		TableName:     resolvedSchema.TableName,
		FieldResolver: a.FieldResolver,
		Fields:        fields,
	}, nil
}
