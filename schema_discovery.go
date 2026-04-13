package limen

import (
	"fmt"
)

// discoverSchemas loads schemas from core configuration and all registered plugins,
// merges plugin-extended fields into core schemas, and returns the resolved map with
// customizations, foreign keys, and indexes applied.
func discoverSchemas(schemaConfig *SchemaConfig, plugins []Plugin) (map[SchemaName]SchemaDefinition, error) {
	schemas := collectCoreSchemas(schemaConfig)
	applyCoreSchemaCustomizations(schemas, schemaConfig)

	for _, plugin := range plugins {
		if err := processPluginSchemas(plugin, schemaConfig, schemas); err != nil {
			return nil, err
		}
	}

	for schemaName, schema := range schemas {
		if err := validateSchemaFields(schema, schemaName, schema.PluginName); err != nil {
			return nil, err
		}
	}

	if err := resolveForeignKeys(schemas); err != nil {
		return nil, err
	}

	if err := resolveIndexes(schemas); err != nil {
		return nil, err
	}

	return schemas, nil
}

func collectCoreSchemas(schemaConfig *SchemaConfig) map[SchemaName]SchemaDefinition {
	userDef := schemaConfig.User.Introspect(schemaConfig).(*SchemaDefinition)
	verificationDef := schemaConfig.Verification.Introspect(schemaConfig).(*SchemaDefinition)
	sessionDef := schemaConfig.Session.Introspect(schemaConfig).(*SchemaDefinition)
	rateLimitDef := schemaConfig.RateLimit.Introspect(schemaConfig).(*SchemaDefinition)
	accountDef := schemaConfig.Account.Introspect(schemaConfig).(*SchemaDefinition)

	return map[SchemaName]SchemaDefinition{
		CoreSchemaUsers:         *userDef,
		CoreSchemaVerifications: *verificationDef,
		CoreSchemaSessions:      *sessionDef,
		CoreSchemaRateLimits:    *rateLimitDef,
		CoreSchemaAccounts:      *accountDef,
	}
}

func applySchemaCustomizations(schema *SchemaDefinition, config *PluginSchemaConfig) {
	if config.TableName != "" {
		schema.TableName = config.TableName
	}

	for i, col := range schema.Columns {
		if newName, exists := config.Fields[col.LogicalField]; exists {
			col.Name = newName
			schema.Columns[i] = col
		}
	}
}

func applyCoreSchemaCustomizations(schemas map[SchemaName]SchemaDefinition, config *SchemaConfig) {
	for schemaName, schema := range schemas {
		customConfig, exists := config.coreSchemaCustomizations[schemaName]
		if !exists {
			continue
		}
		applySchemaCustomizations(&schema, &customConfig)
		schemas[schemaName] = schema
	}
}

func applyPluginCustomizations(def *SchemaDefinition, pluginName PluginName, schemaName SchemaName, config *SchemaConfig) {
	schemaConfigs, exists := config.pluginSchemas[pluginName]
	if !exists {
		return
	}
	schemaConfig, exists := schemaConfigs[schemaName]
	if !exists || (schemaConfig.TableName == "" && len(schemaConfig.Fields) == 0) {
		return
	}

	applySchemaCustomizations(def, &schemaConfig)
}

func processPluginSchemas(plugin Plugin, schemaConfig *SchemaConfig, schemas map[SchemaName]SchemaDefinition) error {
	pluginSchemas := plugin.GetSchemas(schemaConfig)
	for _, introspector := range pluginSchemas {
		def := introspector.(*SchemaDefinition)
		def.PluginName = string(plugin.Name())

		applyPluginCustomizations(def, plugin.Name(), def.SchemaName, schemaConfig)

		if def.Extends != "" {
			if err := mergeSchemaExtension(plugin, def, schemas); err != nil {
				return err
			}
			continue
		}

		if err := mergeSchemaTable(plugin, def, def.SchemaName, schemas); err != nil {
			return err
		}
	}

	return nil
}

func mergeSchemaTable(plugin Plugin, def *SchemaDefinition, schemaName SchemaName, schemas map[SchemaName]SchemaDefinition) error {
	if _, exists := schemas[schemaName]; exists {
		return fmt.Errorf("plugin %s defines schema %s which already exists", plugin.Name(), schemaName)
	}

	for _, schema := range schemas {
		schemaTableName := schema.GetTableName()
		defTableName := def.GetTableName()
		if schemaTableName != defTableName {
			continue
		}

		pluginName := schema.PluginName
		if schema.PluginName == "" {
			pluginName = "core"
		}
		return fmt.Errorf("plugin (%s) defines schema %s which conflicts with schema (%s) which has table \"%s\"",
			plugin.Name(), schemaName, pluginName, string(schemaTableName),
		)
	}
	schemas[schemaName] = *def
	return nil
}

func mergeSchemaExtension(plugin Plugin, def *SchemaDefinition, schemas map[SchemaName]SchemaDefinition) error {
	extendsName := def.Extends

	if !isValidCoreSchema(string(extendsName)) {
		return fmt.Errorf("plugin %s extends invalid core schema: %s", plugin.Name(), extendsName)
	}

	coreSchema, exists := schemas[extendsName]
	if !exists {
		return fmt.Errorf("plugin %s extends unknown schema: %s", plugin.Name(), extendsName)
	}

	coreSchema.Columns = append(coreSchema.Columns, def.Columns...)
	coreSchema.Indexes = append(coreSchema.Indexes, def.Indexes...)
	coreSchema.ForeignKeys = append(coreSchema.ForeignKeys, def.ForeignKeys...)
	schemas[extendsName] = coreSchema
	return nil
}

func resolveForeignKeys(schemas map[SchemaName]SchemaDefinition) error {
	for schemaName, schema := range schemas {
		for i := range schema.ForeignKeys {
			fk := &schema.ForeignKeys[i]

			if fk.Name == "" {
				fk.Name = fmt.Sprintf("fk_%s_%s", schemaName, fk.Column)
			}

			referencedSchema, exists := schemas[fk.ReferencedSchema]
			if !exists {
				return fmt.Errorf("schema %s has foreign key referencing unknown schema: %s", schemaName, fk.ReferencedSchema)
			}

			referencedColumn := findColumnByLogicalField(referencedSchema.Columns, fk.ReferencedField)
			if referencedColumn == nil {
				return fmt.Errorf("schema %s has foreign key referencing unknown field %s in schema %s", schemaName, fk.ReferencedField, fk.ReferencedSchema)
			}

			fk.ReferencedSchema = SchemaName(string(referencedSchema.GetTableName()))
			fk.ReferencedField = SchemaField(referencedColumn.Name)

			resolvedColumn := findColumnByLogicalField(schema.Columns, fk.Column)
			if resolvedColumn == nil {
				return fmt.Errorf("schema %s has foreign key with unknown local column %s", schemaName, fk.Column)
			}
			fk.Column = SchemaField(resolvedColumn.Name)
		}

		schemas[schemaName] = schema
	}

	return nil
}

func resolveIndexes(schemas map[SchemaName]SchemaDefinition) error {
	for schemaName, schema := range schemas {
		for i := range schema.Indexes {
			index := &schema.Indexes[i]
			if index.Name == "" {
				index.Name = fmt.Sprintf("idx_%s_%s", schemaName, joinCustomStringSlice(index.Columns, "_"))
			}

			resolvedColumns, err := resolveIndexColumns(schema.Columns, index.Columns)
			if err != nil {
				return fmt.Errorf("failed to resolve index columns for schema %s : %w", schema.SchemaName, err)
			}
			index.Columns = resolvedColumns
		}
		schemas[schemaName] = schema
	}
	return nil
}

func resolveIndexColumns(schemaColumns []ColumnDefinition, indexColumns []SchemaField) ([]SchemaField, error) {
	resolvedColumns := make([]SchemaField, len(indexColumns))
	for i, col := range indexColumns {
		resolvedColumn := findColumnByLogicalField(schemaColumns, col)
		if resolvedColumn == nil {
			return nil, fmt.Errorf("unknown column %s", col)
		}
		resolvedColumns[i] = SchemaField(resolvedColumn.Name)
	}
	return resolvedColumns, nil
}

// validateSchemaFields validates that a schema definition has no internal conflicts
// It checks for duplicate logicalField and name values within the schema
func validateSchemaFields(schema SchemaDefinition, schemaName SchemaName, ownerName string) error {
	if ownerName == "" {
		ownerName = "core"
	}
	logicalFields := make(map[SchemaField]string)
	names := make(map[string]SchemaField)

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
		if col.LogicalField == logicalField {
			return &col
		}
	}
	return nil
}
